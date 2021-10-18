package consensus

import (
	cstype "chainbft_demo/consensus/types"
	"chainbft_demo/state"
	"chainbft_demo/store"
	"chainbft_demo/types"
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/events"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p"
	"sync"
	"time"
)

var (
	DefaultBlockParent = []byte("single unused block")
)

// 临时配置区
const (
	threshold          = 3
	slotTimeOut        = 10 * time.Second  // 两个slot之间的间隔
	immediateTimeOut   = 0 * time.Second   //
	initialSlotTimeout = 100 * time.Second // 节点启动后clock默认超时时间
)

// 共识状态机实现
// 共识协议在此实现
type ConsensusState struct {
	service.BaseService

	config *config.ConsensusConfig

	// 区块执行器
	blockExec state.BlockExecutor

	// 区块存储器
	blockStore store.Store

	// 区块逻辑时钟
	slotClock SlotClock

	// 共识内部状态
	mtx sync.Mutex
	cstype.RoundState
	state     state.State // 最后一个区块提交后的系统状态
	lastState state.State // 倒数第二个区块提交的系统状态

	// 通信管道
	peerMsgQueue     chan msgInfo       // 处理来自其他节点的消息（包含区块、投票）
	internalMsgQueue chan msgInfo       // 内部消息流通的chan，主要是内部的投票、提案
	eventMsgQueue    chan msgInfo       // 内部消息流通的chan，主要是状态机的状态切换的事件chan
	eventSwitch      events.EventSwitch // consensus和reactor之间通信的组件 - 事件模型

	// 方便测试重写逻辑
	decideProposal func() *types.Proposal               // 生成提案的函数
	setProposal    func(proposal *types.Proposal) error // 待定 不知道有啥用

	// slot不一致的临时补救
	futureProposal   map[types.LTime]types.Proposal
	curSlotStartTime time.Time // 当前slot启动的绝对时间
}

type ConsensusOption func(*ConsensusState)

func NewDefaultConsensusState(
	config *config.ConsensusConfig,
	initSlot types.LTime,
	privVal types.PrivValidator,
	Validators *types.ValidatorSet,
	blockExec state.BlockExecutor,
	blockStore store.Store,
	state state.State,
) *ConsensusState {
	cs := NewConsensusState(
		config,
		initSlot,
		blockExec,
		blockStore,
		state,
		SetValidtorSet(Validators),
		SetValidtor(privVal),
	)
	cs.decideProposal = cs.defaultProposal
	cs.setProposal = cs.defaultSetProposal

	return cs
}

func NewConsensusState(
	config *config.ConsensusConfig,
	initSlot types.LTime,
	blockExec state.BlockExecutor,
	blockStore store.Store,
	state state.State,
	options ...ConsensusOption,
) *ConsensusState {
	cs := &ConsensusState{
		config:     config,
		blockExec:  blockExec,
		blockStore: blockStore,
		slotClock:  NewSlotClock(types.LtimeZero),
		RoundState: cstype.RoundState{
			CurSlot:    initSlot,
			LastSlot:   initSlot,
			Step:       cstype.RoundStepWait,
			Validators: types.NewValidatorSet([]*types.Validator{}),
			VoteSet:    cstype.MakeSlotVoteSet(),
		},
		state:            state,
		peerMsgQueue:     make(chan msgInfo),
		internalMsgQueue: make(chan msgInfo),
		eventMsgQueue:    make(chan msgInfo),
		eventSwitch:      events.NewEventSwitch(),
		futureProposal:   make(map[types.LTime]types.Proposal),
	}

	cs.BaseService = *service.NewBaseService(nil, "CONSENSUS", cs)

	for _, opt := range options {
		opt(cs)
	}

	return cs
}

func (cs *ConsensusState) SetLogger(logger log.Logger) {
	cs.Logger = logger
	if cs.slotClock != nil {
		cs.slotClock.SetLogger(logger)
	}
}

func SetValidtor(validator types.PrivValidator) ConsensusOption {
	return func(cs *ConsensusState) {
		// 设置validator index
		if cs.Validators != nil {
			pub, err := validator.GetPubKey()
			if err == nil {
				cs.ValIndex, _ = cs.Validators.GetByAddress(pub.Address())
			}
		}
		cs.PrivVal = validator
	}
}

func SetValidtorSet(validatorSet *types.ValidatorSet) ConsensusOption {
	return func(cs *ConsensusState) {
		cs.Validators = validatorSet
	}
}

func (cs *ConsensusState) OnStart() error {
	go cs.recieveRoutine()
	go cs.recieveEventRoutine()
	cs.Logger.Info("consensus receive rountines started.")
	return nil
}

func (cs *ConsensusState) OnStop() {
	if err := cs.eventSwitch.Stop(); err != nil {
		cs.Logger.Error("failed trying to stop eventSwitch", "error", err)
	}

	if err := cs.slotClock.Stop(); err != nil {
		cs.Logger.Error("failed trying to stop slotTimer", "error", err)
	}

	if err := cs.Stop(); err != nil {
		cs.Logger.Error("failed trying to stop consensusState", "error", err)
	}
	cs.Logger.Info("consensus server stopped.")
}

// receiveRoutine负责接收所有的消息
// 将原始的消息分类，传递给handleMsg
func (cs *ConsensusState) recieveRoutine() {
	cs.Logger.Debug("consensus receive rountine starts.")
	for {
		select {
		case <-cs.Quit():
			cs.Logger.Info("recieveRoute quit.")
			return

		case msginfo := <-cs.peerMsgQueue:
			// 接收到其他节点的消息
			cs.handleMsg(msginfo)

		case msginfo := <-cs.internalMsgQueue:
			// 收到内部生成的投票or提案
			cs.handleMsg(msginfo)

		case ti := <-cs.slotClock.Chan():
			cs.Logger.Debug("recieved timeout event", "timeout", ti)
			// 统一处理超时事件，目前只有切换slot的超时事件发生
			cs.handleTimeOut(ti)
		}
	}
}

//recieveEventRoutine 负责处理内部的状态事件，完成状态跃迁
func (cs *ConsensusState) recieveEventRoutine() {
	// event
	for {
		select {
		case <-cs.Quit():
			cs.Logger.Info("recieveEventRoutine quit.")
			return
		case msginfo := <-cs.eventMsgQueue:
			//自己节点产生的消息，其实和peerMsgQueue一致：所以有一个统一的入口
			if err := msginfo.Msg.ValidateBasic(); err != nil {
				cs.Logger.Error("internal event validated failed", "err", err)
				continue
			}
			event := msginfo.Msg.(cstype.RoundEvent)
			cs.handleEvent(event)
		}
	}
}

// handleMsg 根据不同的消息类型进行操作
// BlockMessage
// VoteMessage
// SlotTimeout
func (cs *ConsensusState) handleMsg(mi msgInfo) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	msg, peerID := mi.Msg, mi.PeerID

	switch msg := msg.(type) {
	case *ProposalMessage:
		// 收到新的提案
		// TODO核验提案身份 - slot是否一致、提案人是否正确
		if err := msg.Proposal.ValidteBasic(); err != nil {
			cs.Logger.Error("receive wrong proposal.", "error", err, "proposal", msg.Proposal.Block)
			return
		}

		cs.Logger.Info("receive proposal", "slot", cs.CurSlot, "proposal", msg.Proposal)
		if cs.Proposal == nil || cs.Proposal.BlockState == types.DefaultBlock {
			// 如果不为DefaultBlock 说明节点已经收到一个有效的提案了 不再接受其他提案
			if err := cs.setProposal(msg.Proposal); err != nil {
				cs.Logger.Error("set proposal failed.", "error", err)
				return
			}
			cs.Logger.Info("set proposal success.", "cur", cs.CurSlot, "proposer", cs.Proposer.Address)
		} else {
			cs.Logger.Debug("can not set proposal", "proposal", msg.Proposal.Block)
		}
	case *VoteMessage:
		// 收到新的投票信息 尝试将投票加到合适的slot voteset中
		// 根据added，来决定是否转发该投票，只有正确加入voteset的投票才可以继续转发
		// added为false不代表vote不合法，可能只是已经添加过了
		added, err := cs.TryAddVote(msg.Vote, peerID)
		if err != nil {
			// TODO
			//cs.Logger.Error("add vote failed.", "reason", err, "vote", msg.Vote)
			return
		}
		if added {
			// 向reactor转发该消息
			cs.Logger.Debug("add vote success. preprare to broadcast vote", "vote", msg.Vote)
			cs.eventSwitch.FireEvent(EventNewVote, msg.Vote)
		}
	}

}

// 状态机转移函数
func (cs *ConsensusState) handleEvent(event cstype.RoundEvent) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	cs.Logger.Debug("recieve event", "event", event)

	// 先检查事件是否还有效
	if !cs.CurSlot.Equal(event.Slot) && event.Type != cstype.RoundEventNewSlot {
		cs.Logger.Info("receive expired event.", "event", event)
		return
	}

	switch event.Type {
	case cstype.RoundEventNewSlot:
		cs.enterNewSlot(event.Slot)
	case cstype.RoundEventApply:
		cs.enterApply()
	case cstype.RoundEventPropose:
		cs.enterPropose()
	default:
		cs.Logger.Error("Unhandle Event", "event", event)
	}
}

// handleTimeOut 处理超时事件
// 目前就只有SlotTimeOut事件
func (cs *ConsensusState) handleTimeOut(ti timeoutInfo) {
	switch ti.Step {
	case cstype.RoundStepSlot:
		// 切换到新的SLot
		cs.sendEventMessage(msgInfo{cstype.RoundEvent{cstype.RoundEventNewSlot, ti.Slot}, ""})
	}
}

// enterNewSlot 切换到新的slot
// 由RoundEventSlotEnd事件触发
// 负责触发RoundEventApply事件
func (cs *ConsensusState) enterNewSlot(slot types.LTime) {
	defer func() {
		// 成功切换到新的slot，更新状态到RoundStepSLot
		cs.updateStep(cstype.RoundStepSlot)
	}()
	cs.Logger.Info("enter new slot", "slot", slot)

	// 完成切换到slot ？是否需要先更新状态机的状态，如将状态暂时保存起来等
	cs.Logger.Debug("current slot", "slot", cs.slotClock.GetSlot())
	cs.LastSlot = cs.CurSlot
	cs.CurSlot = cs.slotClock.GetSlot()
	cs.curSlotStartTime = time.Now()

	// 计算这一轮的proposer
	cs.decideProposer()

	// 如果切换成功，首先应该重新启动定时器
	cs.slotClock.ResetClock(slotTimeOut)

	// 关于状态机的切换，是直接在这里调用下一轮的函数；
	// 还是在统一的处理函数如handleStateMsg，然后根据不同的消息类型调用不同的阶段函数
	cs.sendEventMessage(msgInfo{cstype.RoundEvent{cstype.RoundEventApply, cs.CurSlot}, ""})
}

// enterApply 进入Apply阶段
// 在这里决定是否确定接受上一个Slot的区块，同时会更新之前的pre-commit block
// RoundEventApply事件触发
// 负责触发RoundEventPropose事件
func (cs *ConsensusState) enterApply() {
	cs.Logger.Info("enter apply step", "slot", cs.CurSlot)
	defer func() {
		if cs.Step != cstype.RoundStepSlot {
			return
		}

		// 设置空提案 该提案不follow任何一个区块
		cs.Proposal = types.MakeEmptyProposal()
		// TODO start timne
		cs.Proposal.Fill(cs.state.ChainID, cs.CurSlot, types.DefaultBlock, DefaultBlockParent, cs.Proposer.Address, cs.state.Validators.Hash(), cs.curSlotStartTime)

		// 成功执行完apply，更新状态到RoundStepApply
		cs.updateStep(cstype.RoundStepApply)

		// 生成切换到propose阶段的事件
		cs.sendEventMessage(msgInfo{cstype.RoundEvent{cstype.RoundEventPropose, cs.CurSlot}, ""})
		// 可能在之前的slot收到当前slot的proposal。原因是因为当前节点是慢节点
		cs.triggleFutureProposal(cs.CurSlot)
	}()

	// 必须处于RoundStepSlot才可以Apply
	if cs.Step != cstype.RoundStepSlot {
		panic(fmt.Sprintf("wrong step, excepted: %v, actul: %v", cstype.RoundStepSlot, cs.Step))
	}

	if !cs.hasProposal() {
		// 如果节点在这一轮slot没有收到提案，提前结束apply阶段
		cs.Logger.Info("proposal is default proposal, skip apply step")
		return
	}

	// 尝试根据目前的投票生成quorum
	voteset := cs.VoteSet.GetVotesBySlot(cs.LastSlot)
	if voteset != nil {
		cs.Logger.Debug("voteset is not empty", "size", len(voteset.GetVotes()), "vote set", voteset.GetVotes())
		// 有收到投票
		witness := cs.Proposal.Block
		quorum := voteset.TryGenQuorum(threshold)
		quorum.SLot = cs.Proposal.Slot

		cs.Logger.Debug("generate quorum done.", "quorum", quorum)
		cs.Proposal.Block.VoteQuorum = quorum

		if quorum.Type == types.SupportQuorum {
			cs.Proposal.Block.BlockState = types.PrecommitBlock

			// 区块转为precommit block，为block里面的support-quorum指向的区块生成commit
			if cs.Proposal.Evidences != nil {
				for _, evidence := range cs.Proposal.Evidences {
					if evidence.Type != types.SupportQuorum {
						continue
					}

					block := cs.state.UnCommitBlocks.QueryBlockByHash(evidence.BlockHash)
					if block == nil {
						// 没有查到区块 可能已经提交
						continue
					}
					block.Commit.SetQuorum(&evidence)
					block.Commit.SetWitness(witness)
				}
			}
		} else if quorum.Type == types.AgainstQuorum {
			cs.Proposal.BlockState = types.ErrorBlock
		} else {
			cs.Proposal.BlockState = types.SuspectBlock
		}
	} else {
		// 没有收到一张投票
		// TODO 是否接受区块待定
		cs.Logger.Debug("proposal receive no votes from other peers", "proposal", cs.Proposal)
		cs.Proposal.BlockState = types.SuspectBlock
	}

	cs.Logger.Debug("proposal ready to commit", "status", cs.Proposal.BlockState)

	// 函数会和blockExec交互
	// 决定提交哪些区块
	stateCopy := cs.state.Copy()
	newState, err := cs.blockExec.ApplyBlock(stateCopy, cs.RoundState.Proposal.Block)
	if err != nil {
		cs.Logger.Error("Apply block failed.", "slot", cs.CurSlot, "reason", err)
		return
	}

	// apply 成功，变更状态
	cs.lastState = cs.state
	cs.state = newState
	cs.Logger.Debug("apply done", "last state", cs.lastState, "current state", cs.state)

	// 尝试校正slot时间
	cs.reviseSlotTime()
}

// hasProposal 判断是否在这一轮slot有收到合适的提案
func (cs *ConsensusState) hasProposal() bool {
	if cs.Proposal != nil && cs.Proposal.BlockState != types.DefaultBlock {
		return true
	}
	return false
}

// enterPropose 如果节点是该轮slot的leader，则在这里提出提案并推送给reactor广播
// RoundEventPropose事件触发
// 不用触发任何事件，等待超时事件即可
func (cs *ConsensusState) enterPropose() {
	cs.Logger.Info("enter propose step", "slot", cs.CurSlot)

	defer func() {
		// 必须处于RoundStepSlot才可以Apply
		//if cs.Step != cstype.RoundStepApply {
		//	return
		//}
		// 比较特殊，提案结束后进入RoundStepWait 等待接收消息
		cs.updateStep(cstype.RoundStepWait)
	}()
	// 必须处于RoundStepSlot才可以Apply
	if cs.Step != cstype.RoundStepApply {
		panic(fmt.Sprintf("wrong step, excepted: %v, actul: %v", cstype.RoundStepSlot, cs.Step))
	}

	// 获得当前节点的validator数据
	_, val := cs.Validators.GetByIndex(cs.ValIndex)
	if !cs.isProposer(val) {
		cs.Logger.Debug("I'm not slot leader. proposePhase end.", "cur", cs.CurSlot, "valIndex", cs.ValIndex)
		return
	} else {
		cs.Logger.Info("I'm leader, prepare to propose.", "cur", cs.CurSlot)
	}

	// 使用函数接口来调用propose逻辑，方便测试
	cs.decideProposal()
}

// 判断这个节点是不是这轮slot的 leader
// 如何判断是否是该轮的leader
func (cs *ConsensusState) isProposer(val *types.Validator) bool {
	if val == nil {
		return false
	}

	if cs.Proposer == nil {
		cs.decideProposer()
	}

	if cs.Proposer.PubKey.Equals(val.PubKey) {
		// 如果当前的validator和这一轮的proposer一致，说明他是该轮proposer
		return true
	}

	return false
}

func (cs *ConsensusState) decideProposer() {
	cs.Proposer = cs.Validators.GetProposer(cs.CurSlot)
	cs.Logger.Info(fmt.Sprintf("this slot proposer is %v", cs.Proposer.Address.String()), "cur", cs.CurSlot)
}

// defaultProposal 默认生成提案的函数
func (cs *ConsensusState) defaultProposal() *types.Proposal {
	cs.Logger.Debug("proposer prepare to use block executor to get proposal", "slot", cs.CurSlot)
	// 特殊，提案前先更新状态
	cs.updateStep(cstype.RoundStepPropose)

	// step 2 从mempool中打包没有冲突的交易
	proposal := cs.blockExec.CreateProposal(cs.state, cs.CurSlot)
	proposal.SlotStartTime = cs.curSlotStartTime

	// step 3 生成签名
	if err := cs.PrivVal.SignProposal(cs.state.ChainID, proposal); err != nil {
		cs.Logger.Error("sign proposal failed", "err", err)
		return nil
	}

	// 向reactor传递block
	cs.Logger.Debug("got proposal", "proposal", proposal)

	// DEBUG 让广播慢一点
	time.Sleep(500 * time.Millisecond)

	// 通过内部chan传递到defaultSetproposal函数统一处理
	cs.sendInternalMessage(msgInfo{&ProposalMessage{Proposal: proposal}, ""})

	return proposal
}

// defaultSetProposal 收到该轮slot leader发布的区块，先验证proposal，然后尝试更新support-quorum
func (cs *ConsensusState) defaultSetProposal(proposal *types.Proposal) error {
	needVote := false
	defer func() {
		if !needVote {
			return
		}
		// 判断提案是否符合发布规则
		if cs.state.IsMatch(proposal) {
			// 提案正确且符合提案规则，投赞成票
			cs.signVote(proposal, true)
		} else {
			// 投反对票
			cs.signVote(proposal, false)
		}
	}()

	cs.Logger.Debug("ready to set proposal", "proposal", proposal.Block)

	// 再次验证proposal - 签名、颁发者是否正确
	// 验证提案的slot是否和当前的slot相等
	if !proposal.Slot.Equal(cs.CurSlot) {
		if proposal.Slot.Greater(cs.CurSlot) {
			if err := cs.TryAddFutureProposal(proposal.Slot, *proposal); err != nil {
				return err
			}
			cs.Logger.Debug("receive future proposal, caching it success")
			return errors.New(fmt.Sprintf("receive future proposal. proposal slot: %v, current slot: %v", proposal.Slot, cs.CurSlot))
		}
		return errors.New(fmt.Sprintf("proposal slot is not same, proposal slot: %v, current slot: %v", proposal.Slot, cs.CurSlot))
	}

	// 验证提案人是否正确
	_, val := cs.Validators.GetByAddress(proposal.ValidatorAddr)
	if !cs.isProposer(val) {
		return errors.New(fmt.Sprintf("%v is not this slot leader, expected: %v", val.String(), cs.Proposer.String()))
	}

	// 验证提案的签名
	if !val.PubKey.VerifySignature(types.ProposalSignBytes(cs.state.ChainID, proposal), proposal.Signature) {
		return errors.New("verifying proposal signature failed")
	}

	needVote = true
	// 尝试根据提案中的support-quorum更新到对应的区块上
	cs.state.UpdateState(proposal.Block)

	cs.Proposal = proposal
	//cs.Proposal.BlockState = types.SuspectBlock

	// 接受提案 然后触发事件通知reactor转发
	cs.eventSwitch.FireEvent(EventNewProposal, proposal)

	return nil
}

// signVote 根据isApproved决定投票性质，生成投票并且传递到reactor广播
func (cs *ConsensusState) signVote(proposal *types.Proposal, isApproved bool) {
	votetype := types.SupportVote
	if isApproved == false {
		votetype = types.AgainstVote
	}

	vote := &types.Vote{
		Slot:             cs.CurSlot,
		BlockHash:        proposal.Hash(),
		Type:             votetype,
		Timestamp:        time.Now(),
		ValidatorAddress: cs.state.Validator.Address,
		ValidatorIndex:   cs.ValIndex,
		Signature:        []byte("signature"),
	}

	if err := cs.PrivVal.SignVote(cs.state.ChainID, vote); err != nil {
		cs.Logger.Error("sign vote failed.", "error", err)
		return
	}

	cs.Logger.Debug("proposal vote", "vote", vote)

	// 通过internalChan传递
	cs.sendInternalMessage(msgInfo{&VoteMessage{Vote: vote}, ""})
}

// TryAddVote 收到投票后尝试将投票加到对应的区块上
// 如果返回(true,nil)则添加成功 如果返回(false,err)则投票本身有问题；否则返回(false,nil)说明投票已经添加过了
func (cs *ConsensusState) TryAddVote(vote *types.Vote, peerID p2p.ID) (bool, error) {
	// 验证投票的合法性 - 签名
	valIdx, val := cs.Validators.GetByAddress(vote.ValidatorAddress)
	if valIdx != vote.ValidatorIndex {
		return false, errors.New(fmt.Sprintf("vote validator has wrong index"))
	}

	// 验证投票的签名
	if !val.PubKey.VerifySignature(types.VoteSignBytes(cs.state.ChainID, vote), vote.Signature) {
		return false, errors.New("vote signature error")
	}

	err := cs.VoteSet.AddVote(vote)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (cs *ConsensusState) TryAddFutureProposal(slot types.LTime, proposal types.Proposal) error {
	_, exist := cs.futureProposal[slot]
	if exist {
		return errors.New(fmt.Sprintf("%v slot already exist", slot))
	}

	cs.futureProposal[slot] = proposal

	return nil
}

func (cs *ConsensusState) triggleFutureProposal(slot types.LTime) {
	proposal, exist := cs.futureProposal[slot]
	if !exist {
		return
	}
	cs.sendInternalMessage(msgInfo{&ProposalMessage{Proposal: &proposal}, ""})
}

// reviseSlotTime 根据最后一个commit的block里的slot time来统一校正时间
func (cs *ConsensusState) reviseSlotTime() {
	if cs.state.LastCommitedBlock == nil {
		return
	}

	// 计算当前slot校正后的开始
	slotdiff := cs.CurSlot.Sub(cs.state.LastCommitedBlock.Slot)
	if slotdiff <= 0 {
		cs.Logger.Error("wrong slot diff", "diff", slotdiff)
		return
	}

	// 当前slot的结束时间，用来修正当前新的wait timeout
	// 当前slot的新的开始时间
	slotStartTime := cs.state.LastCommitedBlock.SlotStartTime
	for i := 0; i < slotdiff; i++ {
		slotStartTime = slotStartTime.Add(slotTimeOut)
	}
	slotEndTime := slotStartTime.Add(slotTimeOut)

	cs.Logger.Debug("[test] slot", "curSlot", cs.CurSlot, "baseSlot", cs.state.LastCommitedBlock.Slot, "slotDiff", slotdiff)
	cs.Logger.Debug("[test] time", "curStart", cs.curSlotStartTime, "baseStart", cs.state.LastCommitedBlock.SlotStartTime, "revisedStart", slotStartTime, "revisedEnd", slotEndTime)
	if time.Now().After(slotEndTime) {
		// 如果当前时间已经超过校正后的结束时间 跳过该轮校正
		cs.Logger.Debug("[test] current time is after advised slot end time. pass")
		return
	}
	newTimeout := slotEndTime.Sub(time.Now())
	cs.curSlotStartTime = slotStartTime
	cs.slotClock.ResetClock(newTimeout)
	cs.Logger.Info("ready to revise slot time", "old", cs.curSlotStartTime, "new", slotStartTime, "new timeout", newTimeout)
}

func (cs *ConsensusState) updateSlot(slot types.LTime) {
	cs.CurSlot = slot
}

func (cs *ConsensusState) updateStep(step cstype.RoundStepType) {
	//cs.Logger.Debug("update state", "current", cs.Step, "next", step)
	cs.Step = step
}

// send a msg into the receiveRoutine regarding our own proposal, block part, or vote
// 往内部的channel写入event
// 直接写可能会因为ceiveRoutine blocked从而导致本协程block
func (cs *ConsensusState) sendEventMessage(mi msgInfo) {
	select {
	case cs.eventMsgQueue <- mi:
	default:
		// NOTE: using the go-routine means our votes can
		// be processed out of order.
		go func() { cs.eventMsgQueue <- mi }()
	}
}

// send a msg into the receiveRoutine regarding our own proposal, block part, or vote
// 直接写可能会因为ceiveRoutine blocked从而导致本协程block
func (cs *ConsensusState) sendInternalMessage(mi msgInfo) {
	select {
	case cs.internalMsgQueue <- mi:
	default:
		// NOTE: using the go-routine means our votes can
		// be processed out of order.
		cs.Logger.Debug("internal msg queue is full; using a go-routine")
		go func() {
			cs.internalMsgQueue <- mi
			cs.Logger.Debug("msgQueue-sending go-routine finished")
		}()
	}
}

// ----- MsgInfo -----
// 与reactor之间通信的消息格式
type msgInfo struct {
	Msg    Message
	PeerID p2p.ID
}

// internally generated messages which may update the state
type timeoutInfo struct {
	Duration time.Duration        `json:"slotTimeOut"`
	Slot     types.LTime          `json:"slot"`
	Step     cstype.RoundStepType `json:"step"`
}

func (ti *timeoutInfo) String() string {
	return fmt.Sprintf("%v ; %d/%v", ti.Duration, ti.Slot, ti.Step)
}

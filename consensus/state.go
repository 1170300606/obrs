package consensus

import (
	cstype "chainbft_demo/consensus/types"
	"chainbft_demo/state"
	"chainbft_demo/store"
	"chainbft_demo/types"
	"fmt"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p"
	tmtype "github.com/tendermint/tendermint/types"
	"sync"
	"time"
)

var (
	DefaultBlockParent = []byte("single unused block")
)

// 临时配置区
const (
	threshold          = 3
	slotTimeOut        = 5 * time.Second   // 两个slot之间的间隔
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

	// !state changes may be triggered by: msgs from peers,
	// msgs from ourself, or by timeouts

	// 通信管道
	peerMsgQueue     chan msgInfo // 处理来自其他节点的消息（包含区块、投票）
	internalMsgQueue chan msgInfo // 内部消息流通的chan，主要是内部的投票、提案
	eventMsgQueue    chan msgInfo // 内部消息流通的chan，主要是状态机的状态切换的事件chan
	reactor          *Reactor     // 用来往其他交易广播消息的接口，在这里是否引入事件模型 - 来简化通信的复杂

	// 方便测试重写逻辑
	decideProposal func() *types.Proposal               // 生成提案的函数
	setProposal    func(proposal *types.Proposal) error // 待定 不知道有啥用
}

type ConsensusOption func(*ConsensusState)

func NewDefaultConsensusState(
	config *config.ConsensusConfig,
	initSlot types.LTime,
	blockExec state.BlockExecutor,
	blockStore store.Store,
	state state.State,
	options ...ConsensusOption,
) *ConsensusState {
	cs := NewConsensusState(
		config,
		initSlot,
		blockExec,
		blockStore,
		state,
		options...)
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
			Validator:  nil,
			Validators: tmtype.NewValidatorSet([]*tmtype.Validator{}),
			Proposal:   nil,
			VoteSet:    cstype.MakeSlotVoteSet(),
		},
		state:            state,
		peerMsgQueue:     make(chan msgInfo),
		internalMsgQueue: make(chan msgInfo),
		eventMsgQueue:    make(chan msgInfo),
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
	if cs.blockExec != nil {
		cs.blockExec.SetLogger(logger)
	}
}

func SetReactor(reactor *Reactor) ConsensusOption {
	return func(cs *ConsensusState) {
		cs.reactor = reactor
	}
}

func SetValidtor(validator tmtype.PrivValidator) ConsensusOption {
	return func(cs *ConsensusState) {
		cs.Validator = validator
	}
}

func SetValidtorSet(validatorSet *tmtype.ValidatorSet) ConsensusOption {
	return func(cs *ConsensusState) {
		cs.Validators = validatorSet
	}
}

func (cs *ConsensusState) OnStart() error {
	go cs.recieveRoutine()
	go cs.recieveEventRoutine()
	return nil
}

func (cs *ConsensusState) OnStop() {
}

// receiveRoutine负责接收所有的消息
// 将原始的消息分类，传递给handleMsg
func (cs *ConsensusState) recieveRoutine() {
	cs.Logger.Debug("consensus receive rountine starts.")
	for {
		select {
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
		case <-cs.Quit():
			cs.Logger.Debug("recieveRoute quit.")
			return
		}
	}
}

//recieveEventRoutine 负责处理内部的状态事件，完成状态跃迁
func (cs *ConsensusState) recieveEventRoutine() {
	// event
	for {
		select {
		case msginfo := <-cs.eventMsgQueue:
			//自己节点产生的消息，其实和peerMsgQueue一致：所以有一个统一的入口
			if err := msginfo.Msg.ValidateBasic(); err != nil {
				cs.Logger.Error("internal event validated failed", "err", err)
				continue
			}
			event := msginfo.Msg.(cstype.RoundEvent)
			cs.handleEvent(event)
		case <-cs.Quit():
			cs.Logger.Debug("recieveEventRoutine quit.")
			return
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
		//if !msg.Proposal.Slot.Equal(cs.CurSlot){
		//	//
		//	return
		//}
		if err := msg.Proposal.ValidteBasic(); err != nil {
			cs.Logger.Error("receive wrong proposal.", "error", err)
			return
		}

		cs.Logger.Debug("receive proposal", "slot", cs.CurSlot, "proposal", msg.Proposal)
		cs.setProposal(msg.Proposal)
	case *VoteMessage:
		// 收到新的投票信息 尝试将投票加到合适的slot voteset中
		// 根据added，来决定是否转发该投票，只有正确加入voteset的投票才可以继续转发
		// added为false不代表vote不合法，可能只是已经添加过了
		added, err := cs.TryAddVote(msg.Vote, peerID)
		if err != nil {
			cs.Logger.Error("add vote failed.", "reason", err, "vote", msg.Vote)
			return
		}
		if added {
			// 向reactor转发该消息
			cs.reactor.BroadcastVote(msg.Vote)
		}
	}

}

// 状态机转移函数
func (cs *ConsensusState) handleEvent(event cstype.RoundEvent) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	cs.Logger.Debug("recieve event", "event", event)
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
	cs.Logger.Debug("enter new slot", "slot", slot)

	// 完成切换到slot ？是否需要先更新状态机的状态，如将状态暂时保存起来等
	cs.Logger.Debug("current slot", "slot", cs.slotClock.GetSlot())
	cs.LastSlot = cs.CurSlot
	cs.CurSlot = cs.slotClock.GetSlot()

	// 设置空提案 该提案不follow任何一个区块
	cs.Proposal = types.MakeEmptyProposal()
	cs.Proposal.Fill(cs.state.ChainID, cs.CurSlot, types.DefaultBlock, DefaultBlockParent, cs.state.Validators.Hash())

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
	cs.Logger.Debug("enter apply step", "slot", cs.CurSlot)
	defer func() {
		// 成功执行完apply，更新状态到RoundStepApply
		cs.updateStep(cstype.RoundStepApply)
	}()

	// TODO 该如何检查slot
	// 必须处于RoundStepSlot才可以Apply
	if cs.Step != cstype.RoundStepSlot {
		panic(fmt.Sprintf("wrong step, excepted: %v, actul: %v", cstype.RoundStepSlot, cs.Step))
	}

	// 尝试根据目前的投票生成quorum
	voteset := cs.VoteSet.GetVotesBySlot(cs.LastSlot)
	if voteset != nil {
		// 有收到投票
		witness := cs.Proposal.Block
		quorum := voteset.TryGenQuorum(threshold)
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
}

// enterPropose 如果节点是该轮slot的leader，则在这里提出提案并推送给reactor广播
// RoundEventPropose事件触发
// 不用触发任何事件，等待超时事件即可
func (cs *ConsensusState) enterPropose() {
	cs.Logger.Debug("enter propose step", "slot", cs.CurSlot)

	defer func() {
		// 比较特殊，提案结束后进入RoundStepWait 等待接收消息
		cs.updateStep(cstype.RoundStepWait)
	}()

	// TODO 该如何检查slot
	// 必须处于RoundStepSlot才可以Apply
	if cs.Step != cstype.RoundStepApply {
		panic(fmt.Sprintf("wrong step, excepted: %v, actul: %v", cstype.RoundStepApply, cs.Step))
	}
	if !cs.isProposer() {
		cs.Logger.Debug("I'm not slot leader. proposePhase end.")
		return
	}

	// 使用函数接口来调用propose逻辑，方便测试
	cs.decideProposal()
}

// 判断这个节点是不是这轮slot的 leader
// TODO 如何判断是否是该轮的leader
func (cs *ConsensusState) isProposer() bool {
	return true
}

// defaultProposal 默认生成提案的函数
func (cs *ConsensusState) defaultProposal() *types.Proposal {
	cs.Logger.Debug("proposer prepare to use block executor to get proposal", "slot", cs.CurSlot)
	// 特殊，提案前先更新状态
	cs.updateStep(cstype.RoundStepPropose)

	// step 2 从mempool中打包没有冲突的交易
	proposal := cs.blockExec.CreateProposal(cs.state, cs.CurSlot)

	// 向reactor传递block
	cs.Logger.Debug("got proposal", "proposal", proposal)
	// 通过内部chan传递到defaultSetproposal函数统一处理
	cs.sendInternalMessage(msgInfo{&ProposalMessage{Proposal: proposal}, ""})

	return proposal
}

// defaultSetProposal 收到该轮slot leader发布的区块，先验证proposal，然后尝试更新support-quorum
func (cs *ConsensusState) defaultSetProposal(proposal *types.Proposal) error {
	var err error
	defer func() {
		if err == nil {
			// 提案正确且符合提案规则，投赞成票
			cs.signVote(proposal, true)
		} else {
			// 投反对票
			cs.signVote(proposal, false)
		}
	}()

	// TODO 再次验证proposal - 签名、颁发者是否正确、提案是否符合提案规则

	// 尝试根据提案中的support-quorum更新到对应的区块上
	cs.state.UpdateState(proposal.Block)

	cs.Proposal = proposal

	// 接受提案 然后转发
	cs.reactor.BroadcastProposal(proposal)

	return nil
}

// signVote 根据isApproved决定投票性质，生成投票并且传递到reactor广播
func (cs *ConsensusState) signVote(proposal *types.Proposal, isApproved bool) {
	// TODO 判断提案是否符合发布规则
	votetype := types.SupportVote
	if isApproved == false {
		votetype = types.AgainstVote
	}

	validatorPubKey, err := cs.Validator.GetPubKey()

	if err != nil {
		return
	}
	vote := &types.Vote{
		Slot:             cs.CurSlot,
		BlockHash:        proposal.Hash(),
		Type:             votetype,
		Timestamp:        time.Now(),
		ValidatorAddress: types.GetAddress(validatorPubKey),
		ValidatorIndex:   -1,
		Signature:        []byte("signature"),
	}

	cs.Logger.Debug("proposal vote", "vote", vote)

	// 通过internalChan传递
	cs.sendInternalMessage(msgInfo{&VoteMessage{Vote: vote}, ""})
}

// TryAddVote 收到投票后尝试将投票加到对应的区块上
// 如果返回(true,nil)则添加成功 如果返回(false,err)则投票本身有问题；否则返回(false,nil)说明投票已经添加过了
func (cs *ConsensusState) TryAddVote(vote *types.Vote, peerID p2p.ID) (bool, error) {
	// TODO 验证投票的合法性 - 签名

	err := cs.VoteSet.AddVote(vote)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (cs *ConsensusState) updateSlot(slot types.LTime) {
	cs.CurSlot = slot
}

func (cs *ConsensusState) updateStep(step cstype.RoundStepType) {
	cs.Logger.Debug("update state", "current", cs.Step, "next", step)
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
		cs.Logger.Debug("internal msg queue is full; using a go-routine")
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
		go func() { cs.internalMsgQueue <- mi }()
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

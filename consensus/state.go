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
	"sync"
	"time"
)

// 临时配置区
const (
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
	state state.State // 最后一个区块提交后的系统状态

	// !state changes may be triggered by: msgs from peers,
	// msgs from ourself, or by timeouts

	// 通信管道
	peerMsgQueue     chan msgInfo // 处理来自其他节点的消息（包含区块、投票）
	internalMsgQueue chan msgInfo // 内部消息流通的chan，主要是状态机的状态切换的事件chan
	reactor          *Reactor     // 用来往其他交易广播消息的接口，在这里是否引入事件模型 - 来简化通信的复杂

	// 方便测试重写逻辑
	decideProposal func()                               // 生成提案的函数
	setProposal    func(proposal *types.Proposal) error // 待定 不知道有啥用
}

type ConsensusOption func(*ConsensusState)

func NewDefaultConsensusState(
	config *config.ConsensusConfig,
	blockExec state.BlockExecutor,
	blockStore store.Store,
	state state.State,
	options ...ConsensusOption,
) *ConsensusState {
	cs := NewConsensusState(
		config,
		blockExec,
		blockStore,
		state,
		options...)
	cs.decideProposal = cs.defaultProposal

	return cs
}

func NewConsensusState(
	config *config.ConsensusConfig,
	blockExec state.BlockExecutor,
	blockStore store.Store,
	state state.State,
	options ...ConsensusOption,
) *ConsensusState {
	cs := &ConsensusState{
		config:           config,
		blockExec:        blockExec,
		blockStore:       blockStore,
		slotClock:        NewSlotClock(types.LtimeZero),
		RoundState:       cstype.RoundState{},
		state:            state,
		peerMsgQueue:     make(chan msgInfo),
		internalMsgQueue: make(chan msgInfo),
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
		case msginfo := <-cs.internalMsgQueue:
			//自己节点产生的消息，其实和peerMsgQueue一致：所以有一个统一的入口
			if err := msginfo.msg.ValidateBasic(); err != nil {
				cs.Logger.Error("internal event validated failed", "err", err)
				continue
			}
			event := msginfo.msg.(cstype.RoundEvent)
			cs.handleEvent(event)
		}
	}
}

// handleMsg 根据不同的消息类型进行操作
// BlockMessage
// VoteMessage
// SlotTimeout
func (cs *ConsensusState) handleMsg(msg msgInfo) {
}

// 状态机转移函数
func (cs *ConsensusState) handleEvent(event cstype.RoundEvent) {
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
		// TODO 切换到新的SLot
		cs.sendInternalMessage(msgInfo{cstype.RoundEvent{cstype.RoundEventNewSlot, ti.Slot}, ""})
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

	// TODO 完成切换到slot 首先更新状态机的状态，如将状态暂时保存起来等
	cs.Logger.Debug("current slot", "slot", cs.slotClock.GetSlot())
	cs.CurSlot = cs.slotClock.GetSlot()

	// 如果切换成功，首先应该重新启动定时器
	cs.slotClock.ResetClock(slotTimeOut)

	// 关于状态机的切换，是直接在这里调用下一轮的函数；
	// 还是在统一的处理函数如handleStateMsg，然后根据不同的消息类型调用不同的阶段函数
	cs.sendInternalMessage(msgInfo{cstype.RoundEvent{cstype.RoundEventApply, cs.CurSlot}, ""})
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

	// 函数会和blockExec交互
	// 决定提交哪些区块
	//cs.blockExec.ApplyBlock(cs.RoundState.Proposal.Block)
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
		return
	}

	// 使用函数接口来调用propose逻辑，方便测试
	cs.decideProposal()
}

// 判断这个节点是不是这轮slot的 leader
func (cs *ConsensusState) isProposer() bool {
	return true
}

func (cs *ConsensusState) defaultProposal() {
	cs.Logger.Debug("proposer prepare to use block executor to get proposal", "slot", cs.CurSlot)
	// 特殊，提案前先更新状态
	cs.updateStep(cstype.RoundStepPropose)

	// step 1 根据state中的信息，选出下一轮应该follow哪个区块
	_ = cs.state.NewBranch()

	// step 2 从mempool中打包没有冲突的交易
	proposal := cs.blockExec.CreateProposal(cs.state, cs.CurSlot)

	// step 3 填补proposal中的字段 完成打包阶段
	proposal.Fill()

	//  向reactor传递block
	cs.Logger.Debug("got proposal", "proposal", proposal)
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
func (cs *ConsensusState) sendInternalMessage(mi msgInfo) {
	select {
	case cs.internalMsgQueue <- mi:
	default:
		// NOTE: using the go-routine means our votes can
		// be processed out of order.
		// TODO: use CList here for strict determinism and
		// attempt push to internalMsgQueue in receiveRoutine
		cs.Logger.Debug("internal msg queue is full; using a go-routine")
		go func() { cs.internalMsgQueue <- mi }()
	}
}

// ----- MsgInfo -----
// 与reactor之间通信的消息格式
type msgInfo struct {
	msg    Message
	peerId p2p.ID
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

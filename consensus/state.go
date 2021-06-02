package consensus

import (
	cstype "chainbft_demo/consensus/types"
	"chainbft_demo/state"
	"chainbft_demo/store"
	"chainbft_demo/types"
	"fmt"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p"
	"sync"
	"time"
)

// 临时配置区
const (
	slotTimeOut      = 10 * time.Second
	immediateTimeOut = 0 * time.Second
)

// 共识状态机实现
// 共识协议在此实现
type ConsensusState struct {
	service.BaseService

	// 区块执行器
	blockExec state.BlockExecutor

	// 区块存储器
	blockStore store.Store

	// 区块逻辑时钟
	slotClock Slot

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
	decideProposal func(slot types.LTime)               // 生成提案的函数
	setProposal    func(proposal *types.Proposal) error // 待定 不知道有啥用
}

func (cs *ConsensusState) SetLogger(logger log.Logger) {
	cs.SetLogger(logger)
	cs.slotClock.SetLogger(logger)
	cs.blockExec.SetLogger(logger)
}

func (cs *ConsensusState) OnStart() error {
	go cs.recieveRoutine()
	return nil
}

func (cs *ConsensusState) OnStop() {
}

// receiveRoutine负责接收所有的消息
// 将原始的消息分类，传递给handleMsg
func (cs *ConsensusState) recieveRoutine() {
	for {
		select {
		case msginfo := <-cs.peerMsgQueue:
			// 接收到其他节点的消息
			cs.handleMsg(msginfo)
		case msginfo := <-cs.internalMsgQueue:
			//自己节点产生的消息，其实和peerMsgQueue一致：所以有一个统一的入口
			if err := msginfo.msg.ValidateBasic(); err != nil {
				cs.Logger.Error("internal event validated failed", "err", err)
				continue
			}
			event := msginfo.msg.(cstype.RoundEvent)
			cs.handleEvent(event)
		case <-cs.slotClock.Chan():
			// 统一处理超时事件，目前只有切换slot的超时事件发生
			ti := timeoutInfo{
				Duration: slotTimeOut,
				Slot:     0,
				Step:     0,
			}
			cs.handleTimeOut(ti)
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
		cs.sendInternalMessage(msgInfo{cstype.RoundEvent{cstype.RoundEventNewSlot, cs.Slot}, ""})
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
	cs.Logger.Debug("enter new slot", "slot", cs.Slot)

	// TODO 完成切换到slot 首先更新状态机的状态，如将状态暂时保存起来等

	// 如果切换成功，首先应该重新启动定时器
	cs.slotClock.Reset(slotTimeOut)

	// 关于状态机的切换，是直接在这里调用下一轮的函数；
	// 还是在统一的处理函数如handleStateMsg，然后根据不同的消息类型调用不同的阶段函数
	cs.sendInternalMessage(msgInfo{cstype.RoundEvent{cstype.RoundEventApply, cs.Slot}, ""})
}

// enterApply 进入Apply阶段
// 在这里决定是否确定接受上一个Slot的区块，同时会更新之前的pre-commit block
// RoundEventApply事件触发
// 负责触发RoundEventPropose事件
func (cs *ConsensusState) enterApply() {
	defer func() {
		// 成功执行完apply，更新状态到RoundStepApply
		cs.updateStep(cstype.RoundStepApply)
	}()
	// 函数会和blockExec交互
	// 决定提交哪些区块
	cs.blockExec.CommitBlock(cs.RoundState.Proposal.Block)
}

// enterPropose 如果节点是该轮slot的leader，则在这里提出提案并推送给reactor广播
// RoundEventPropose事件触发
// 不用触发任何事件，等待超时事件即可
func (cs *ConsensusState) enterPropose() {
	defer func() {
		// 比较特殊，提案结束后进入RoundStepWait 等待接收消息
		cs.updateStep(cstype.RoundStepWait)
	}()

	if !cs.isProposer() {
		return
	}

	// 使用函数接口来调用propose逻辑，方便测试
	cs.decideProposal(cs.Slot)
}

// 判断这个节点是不是这轮slot的 leader
func (cs *ConsensusState) isProposer() bool {
	return true
}

func (cs *ConsensusState) defaultPropose() {
	// 特殊，提案前先更新状态
	cs.updateStep(cstype.RoundStepPropose)
	proposal := cs.blockExec.CreateProposal()

	//  向reactor传递block
	cs.Logger.Debug("generate proposal", "proposal", proposal)
}

func (cs *ConsensusState) updateSlot(slot types.LTime) {
	cs.Slot = slot
}

func (cs *ConsensusState) updateStep(step cstype.RoundStepType) {
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

package consensus

import (
	cstype "chainbft_demo/consensus/types"
	"chainbft_demo/slot"
	"chainbft_demo/state"
	"chainbft_demo/store"
	"chainbft_demo/types"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p"
	"sync"
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
	slotClock slot.Slot

	// 共识内部状态
	mtx sync.Mutex
	cstype.RoundState
	state state.State // 最后一个区块提交后的系统状态

	// !state changes may be triggered by: msgs from peers,
	// msgs from ourself, or by timeouts

	// 通信管道
	peerMsgQueue  chan msgInfo // 处理来自其他节点的消息（包含区块、投票）
	internalQueue chan msgInfo // 内部消息流通的chan，主要是状态机的状态切换的事件chan
	reactor       *Reactor     // 用来往其他交易广播消息的接口，在这里是否引入事件模型 - 来简化通信的复杂

	// 方便测试重写逻辑
	decideProposal func(slot types.LTime)
	doPrevote      func(slot types.LTime)
	setProposal    func(proposal *types.Proposal) error
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
		case msginfo := <-cs.internalQueue:
			//自己节点产生的消息，其实和peerMsgQueue一致：所以有一个统一的入口
			cs.handleMsg(msginfo)
		case <-cs.slotClock.GetTimeOutChan():
			// 统一处理超时事件，目前只有切换slot的超时事件发生
			cs.handleTimeOut(msgInfo{})
		}
	}
}

// handleMsg 根据不同的消息类型进行操作
// BlockMessage
// VoteMessage
// SlotTimeout
func (cs *ConsensusState) handleMsg(msg msgInfo) {
}

// handleTimeOut 处理超时事件
// 目前就只有SlotTimeOut事件
func (cs *ConsensusState) handleTimeOut(msg msgInfo) {
}

// ----- MsgInfo -----
type msgInfo struct {
	msg    Message
	peerId p2p.ID
}

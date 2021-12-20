package types

import (
	"chainbft_demo/types"
	"fmt"
)

//-----------------------------------------------------------------------------
// RoundStepType enum type

// RoundStepType enumerates the state of the consensus state machine
type RoundStepType uint8

// RoundStepType
const (
	RoundStepSlot    = RoundStepType(0x01) // 切换到新slot的事件，目前只有SlotClock的timeout事件可以触发
	RoundStepApply   = RoundStepType(0x02) // 成功切换到新的slot后
	RoundStepPropose = RoundStepType(0x03) // Apply阶段结束后
	RoundStepWait    = RoundStepType(0x04) // 发生SlotTimeOut事件
)

func (step RoundStepType) String() string {
	switch step {
	case RoundStepSlot:
		return "RoundStepSlot"
	case RoundStepApply:
		return "RoundStepApply"
	case RoundStepPropose:
		return "RoundStepPropose"
	case RoundStepWait:
		return "RoundStepWait"
	default:
		return "RoundStepUnkonwn"
	}
}

func (step RoundStepType) ValidateBasic() error {
	// 验证类型是否合法
	return nil
}

// RoundEventType 列举了状态机所有的事件
type RoundEventType uint8

const (
	RoundEventNewSlot = RoundEventType(0x01) // slot超时的事件
	RoundEventApply   = RoundEventType(0x02) // 进入apply step
	RoundEventPropose = RoundEventType(0x03) // 进入proposal step
	//RoundEventWait    = RoundEventType(0x04) // 进入Wait step
	//RoundEventWait = RoundEventType(0x03) // 进入proposal step
)

func (re RoundEventType) String() string {
	switch re {
	case RoundEventNewSlot:
		return "RoundEventNewSlot"
	case RoundEventApply:
		return "RoundEventApply"
	case RoundEventPropose:
		return "RoundEventPropose"
	default:
		return "RoundEventUnkown"
	}
}

func (step RoundEventType) ValidateBasic() error {
	// 验证类型是否合法
	return nil
}

type RoundEvent struct {
	Type RoundEventType
	Slot types.LTime
}

func (event RoundEvent) String() string {
	return fmt.Sprintf("{Round Event: %v/%v}", event.Slot, event.Type)
}

func (event RoundEvent) ValidateBasic() error {
	return event.Type.ValidateBasic()
}

// definitions of CHAIN_BFT Consensus state machine
type RoundState struct {

	// 基础的共识信息
	CurSlot    types.LTime
	LastSlot   types.LTime
	Step       RoundStepType
	ValIndex   int32
	PrivVal    types.PrivValidator // 验证者的信息 - 私钥，用来签名
	Validators *types.ValidatorSet // 目前共识中的所有的验证者集合

	Proposer *types.Validator // 这一轮的提案者
	Proposal *types.Proposal  // 这一轮收到的合法提案
	VoteSet  *SlotVoteSet     // slot=> voteSet的投票集合

	LastCommit *types.Commit // 最后一个 committed区块的commit
}

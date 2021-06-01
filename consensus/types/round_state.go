package types

import (
	types "chainbft_demo/types"
	tmtype "github.com/tendermint/tendermint/types"
)

//-----------------------------------------------------------------------------
// RoundStepType enum type

// RoundStepType enumerates the state of the consensus state machine
type RoundStepType uint8

// RoundStepType
const (
	RoundStepSlot    = RoundStepType(0x01) //
	RoundStepApply   = RoundStepType(0x02) // 成功切换到新的slot后
	RoundStepPropose = RoundStepType(0x03) // Apply阶段结束后
	RoundStepWait    = RoundStepType(0x04) // 发生SlotTimeOut事件
)

// definitions of CHAIN_BFT Consensus state machine
type RoundState struct {

	// 基础的共识信息
	Slot       int64
	Step       RoundStepType
	Validator  *tmtype.PrivValidator // 验证者的信息 - 私钥，用来签名
	Validators *tmtype.ValidatorSet  // 目前共识中的所有的验证者集合

	Proposal *types.Proposal // 这一轮收到的合法提案
	VoteSet  types.Vote      // 这一轮的投票集合

	LastCommit *types.Commit // 最后一个 committed区块的commit
}

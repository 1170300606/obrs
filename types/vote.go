package types

import (
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"time"
)

type VoteType uint8

const (
	SupportVote = VoteType(1)
	AgainstVote = VoteType(2)
)

func (t VoteType) String() string {
	switch t {
	case SupportVote:
		return "SupportVote"
	case AgainstVote:
		return "AgainstVote"
	default:
		return "UnkownVote"
	}
}

// Vote - 针对某个提案的单个投票，一个节点只能在一个slot发布一个vote
type Vote struct {
	Slot             LTime            `json:"slot"`
	BlockHash        tmbytes.HexBytes `json:"block_hash"`
	Type             VoteType         `json:"Vote_type"`
	Timestamp        time.Time        `json:"timestamp"`
	ValidatorAddress Address          `json:"validator_address"`
	ValidatorIndex   int32            `json:"validator_index"`
	Signature        tmbytes.HexBytes `json:"signature"`
}

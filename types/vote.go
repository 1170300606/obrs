package types

import (
	"bytes"
	"github.com/tendermint/tendermint/crypto/merkle"
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

// Equal 并不是判断两个vote是否一样，而是判断这两个投票的人和发布的时间是否一致
func (v *Vote) Equal(other *Vote) bool {
	if other == nil || v == nil {
		return false
	}

	// 同一个人在同一个slot即为同一个投票
	return v.Slot.Equal(other.Slot) &&
		bytes.Equal(v.ValidatorAddress, other.ValidatorAddress)
}

// 签名的内容
func VoteSignBytes(chainID string, vote *Vote) []byte {
	return merkle.HashFromByteSlices([][]byte{
		[]byte(chainID),
		vote.Slot.Hash(),
		[]byte(vote.Type.String()),
		vote.ValidatorAddress,
		vote.BlockHash,
	})
}

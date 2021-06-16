package types

import tmbytes "github.com/tendermint/tendermint/libs/bytes"

type QuorumType uint8

const (
	EmptyQuorum   = QuorumType(0)
	SupportQuorum = QuorumType(1)
	AgainstQuorum = QuorumType(2)
)

func (q QuorumType) String() string {
	switch q {
	case EmptyQuorum:
		return "EmptyQuorum"
	case SupportQuorum:
		return "SupportQuorum"
	case AgainstQuorum:
		return "AgainstQuorum"
	default:
		return "UnkownQuorum"
	}
}

func NewQuorum() Quorum {
	return Quorum{
		Type:      0,
		Signature: nil,
	}
}

// Quorum明确的表示某个区块有2t+1个节点赞同或反对
type Quorum struct {
	SLot      LTime
	BlockHash tmbytes.HexBytes
	Type      QuorumType
	Signature tmbytes.HexBytes
}

func (q *Quorum) IsEmpty() bool {
	if q.Type == EmptyQuorum {
		return true
	}
	return false
}

func (q *Quorum) Copy() Quorum {
	newQ := Quorum{
		SLot:      q.SLot,
		BlockHash: make([]byte, len(q.BlockHash)),
		Type:      q.Type,
		Signature: make([]byte, len(q.Signature)),
	}
	copy(newQ.BlockHash, q.BlockHash)
	copy(newQ.Signature, q.Signature)
	return newQ
}

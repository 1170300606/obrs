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

// 由2t+1vote组成
type Quorum struct {
	Type      QuorumType
	Signature tmbytes.HexBytes
}

func (q *Quorum) IsEmpty() bool {
	if q.Type == EmptyQuorum {
		return true
	}
	return false
}

// TODO 将votes还原成聚合签名
func NewQuorum(votes []*Vote) Quorum {
	return Quorum{
		Type:      0,
		Signature: nil,
	}
}

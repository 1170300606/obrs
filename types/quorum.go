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
func (q *Quorum) Generate(votes []Vote, qtype QuorumType) error {
	q.Signature = []byte("aggregated signature")
	q.Type = qtype

	return nil
}

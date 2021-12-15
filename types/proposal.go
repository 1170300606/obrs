package types

import "time"

type Proposal struct {
	// 基本的提案信息
	*Block `json:"block"`

	// for DEBUG
	SendTime    time.Time `json:"send_time"`
	Round       int       `json:"gossip_round"`
	From        int       `json:"from"`
	ReceiveTime time.Time
	MBSize      float64
}

func ProposalSignBytes(chainID string, p *Proposal) []byte {
	return p.Block.Hash()
}

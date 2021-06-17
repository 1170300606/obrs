package types

type Proposal struct {
	// 基本的提案信息
	*Block `json:"block"`
}

func ProposalSignBytes(chainID string, p *Proposal) []byte {
	return p.Block.Hash()
}

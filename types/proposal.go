package types

type Proposal struct {
	// 基本的提案信息
	ChainID string `json:"chain_id"`
	Slot    LTime  `json:"slot"`
	*Block  `json:"block"`
}

func (proposal *Proposal) Fill() {

}

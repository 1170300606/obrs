package types

type Proposal struct {
	// 基本的提案信息
	ChainID string `json:"chain_id"`
	Slot    int64  `json:"slot"`
	Block   `json:"block"`
}

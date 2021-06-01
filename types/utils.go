package types

// MakeBlock 返回一个头信息为空区块
func MakeBlock(height int64, txs []Tx, lastCommit *Commit) *Block {
	block := &Block{
		Header: Header{
			ChainID:        "",
			Slot:           0,
			BlockState:     0,
			LastBlockHash:  nil,
			TxsHash:        nil,
			ResultHash:     nil,
			ValidatorsHash: nil,
			BlockHash:      nil,
		},
		Data: Data{
			Txs: txs,
		},
		Quorum:   Quorum{},
		Evidence: lastCommit,
	}
	return block
}

package types

// MakeBlock 返回一个头信息为空区块
func MakeBlock(chainID string, slot LTime, txs []Tx) *Block {
	block := &Block{
		Header: Header{
			ChainID: chainID,
			Slot:    slot,
		},
		Data: Data{
			Txs: txs,
		},
	}
	return block
}

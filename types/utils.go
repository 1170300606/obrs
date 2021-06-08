package types

// MakeBlock 返回一个头信息为空区块
func MakeBlock(txs []Tx) *Block {
	block := &Block{
		Header: Header{},
		Data: Data{
			Txs: txs,
		},
	}
	return block
}

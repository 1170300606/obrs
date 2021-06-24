package types

func MakeGenesisBlock(ChainID string, supportQuorum Quorum) *Block {
	return &Block{
		Header: Header{
			ChainID:       ChainID,
			Slot:          LtimeZero,
			BlockState:    CommiitedBlock,
			LastBlockHash: []byte{},
			BlockHash:     nil,
		},
		Data: Data{
			Txs: Txs{},
		},
		VoteQuorum: supportQuorum,
		Evidences:  []Quorum{},
	}
}

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

func MakeEmptyProposal() *Proposal {
	return &Proposal{
		&Block{
			Header:     Header{},
			Data:       Data{Txs: Txs{}},
			VoteQuorum: Quorum{},
			Evidences:  []Quorum{},
			Commit:     nil,
		},
	}
}

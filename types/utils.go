package types

import "time"

func MakeGenesisBlock(ChainID string, genesisTime time.Time) *Block {
	return &Block{
		Header: Header{
			ChainID:       ChainID,
			Slot:          LtimeZero,
			Ctime:         genesisTime,
			BlockState:    CommiitedBlock,
			LastBlockHash: []byte{},
			BlockHash:     nil,
		},
		Data: Data{
			Txs: Txs{},
		},
		VoteQuorum: Quorum{},
		Evidences:  []Quorum{},
	}
}

// MakeBlock 返回一个头信息为空区块
func MakeBlock(txs []Tx) *Block {
	block := &Block{
		Header: Header{
			Ctime: time.Now(),
		},
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

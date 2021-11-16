package types

import "time"

func MakeGenesisBlock(ChainID string, genesisTime time.Time) *Block {
	return &Block{
		Header: Header{
			ChainID:       ChainID,
			Slot:          LtimeZero,
			ProposalTime:  genesisTime,
			BlockState:    CommittedBlock,
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
			ProposalTime: time.Now(),
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

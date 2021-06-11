package types

import tmbytes "github.com/tendermint/tendermint/libs/bytes"

func MakeGenesisBlock(ChainID string, supportQuorum tmbytes.HexBytes) *Block {
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
		Quorum: Quorum{
			// TODO
			SupportQuorum,
			nil,
		},
		Evidences: Quorum{
			EmptyQuorum,
			supportQuorum,
		},
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
			Header:    Header{},
			Data:      Data{Txs: Txs{}},
			Quorum:    Quorum{},
			Evidences: Quorum{},
			Commit:    nil,
		},
	}
}

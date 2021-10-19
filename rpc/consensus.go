package rpc

import (
	"chainbft_demo/types"
	"github.com/tendermint/tendermint/libs/bytes"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

// CheckTx result
type ResultBlockTree struct {
	Blocks []ResultBlock `json:"blocks"`
}

type ResultBlock struct {
	Slot          types.LTime    `json:"slot"`
	BlockStatus   string         `json:"block_status"`
	LastBlockHash bytes.HexBytes `json:"last_blockhash"`
	BlockHash     bytes.HexBytes `json:"blockhash"`
	TxNum         int            `json:"tx_num"`
	ValidatorAddr bytes.HexBytes `json:"validator_addr"`
}

func BlockTree(ctx *rpctypes.Context) (*ResultBlockTree, error) {
	// 遍历已提交的block
	cons := env.Consensus
	originBlocks := cons.GetAllBlocks()

	blocks := []ResultBlock{}

	for _, oblock := range originBlocks {
		block := ResultBlock{
			Slot:          oblock.Slot,
			BlockStatus:   oblock.BlockState.String(),
			LastBlockHash: oblock.LastBlockHash,
			BlockHash:     oblock.BlockHash,
			TxNum:         len(oblock.Data.Txs),
			ValidatorAddr: oblock.ValidatorAddr,
		}
		blocks = append(blocks, block)
	}

	return &ResultBlockTree{
		Blocks: blocks,
	}, nil
}

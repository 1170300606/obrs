package rpc

import (
	"chainbft_demo/libs/utils"
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

	BlockPrecommitTime float64 `json:"precommit_time"` // 达成precommit状态的耗时
	BlockCommitTime    float64 `json:"commit_time"`    // 达成precommit状态的耗时
	TotalConsensusTime float64 `json:"consensus_time"` // 共识总耗时
	ResultLatency
}

type ResultLatency struct {
	TxNum        int     `json:"tx_num"`
	Max_Latency  float64 `json:"max_tx_latency"`
	Min_Latency  float64 `json:"min_tx_latency"`
	Mean_Latency float64 `json:"mean_tx_latency"`
	Avg_Latency  float64 `json:"avg_tx_latency"`
}

func BlockTree(ctx *rpctypes.Context) (*ResultBlockTree, error) {
	// 遍历已提交的block
	cons := env.Consensus
	originBlocks := cons.GetAllBlocks()

	blocks := []ResultBlock{}

	for _, oblock := range originBlocks {
		oblock.CalculateTime()
		txsLatency := make([]float64, 0, len(oblock.Txs))
		for i := 0; i < len(oblock.Txs); i++ {
			sbtx, _ := oblock.Txs[i].(*types.SmallBankTx)
			txLatency := float64(oblock.BlockCommitTime - sbtx.TxSendTimestamp)
			if txLatency > 0 {
				txsLatency = append(txsLatency, txLatency/1e9)
			}
		}

		block := ResultBlock{
			Slot:          oblock.Slot,
			BlockStatus:   oblock.BlockState.String(),
			LastBlockHash: oblock.LastBlockHash,
			BlockHash:     oblock.BlockHash,
			TxNum:         len(oblock.Data.Txs),
			ValidatorAddr: oblock.ValidatorAddr,

			BlockPrecommitTime: float64(oblock.TimePrecommit) / 1e9,
			BlockCommitTime:    float64(oblock.TimeCommit) / 1e9,
			TotalConsensusTime: float64(oblock.TimeConsensus) / 1e9,
			ResultLatency: ResultLatency{
				TxNum:        len(txsLatency),
				Max_Latency:  utils.Max(txsLatency...),
				Min_Latency:  utils.Min(txsLatency...),
				Mean_Latency: utils.Mean(txsLatency...),
				Avg_Latency:  utils.Avg(txsLatency...),
			},
		}
		blocks = append(blocks, block)
	}

	return &ResultBlockTree{
		Blocks: blocks,
	}, nil
}

func BlockLatency(ctx *rpctypes.Context, slot int) (*ResultLatency, error) {
	return nil, nil
}

package rpc

import (
	meml "chainbft_demo/mempool"
	"chainbft_demo/types"
	"encoding/hex"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

func BroadcastTxAsync(ctx *rpctypes.Context, txHex []byte) (*coretypes.ResultBroadcastTx, error) {
	tx := make([]byte, len(txHex)/2)
	hex.Decode(tx, txHex)
	sbtx := types.Tx{}
	json.Unmarshal(tx, &sbtx)

	err := env.Mempool.CheckTx(sbtx, meml.TxInfo{})

	if err != nil {
		return nil, err
	}
	return &coretypes.ResultBroadcastTx{Hash: sbtx.Hash()}, nil
}

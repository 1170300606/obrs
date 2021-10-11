package rpc

import (
	meml "chainbft_demo/mempool"
	"chainbft_demo/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

func BroadcastTxAsync(ctx *rpctypes.Context, tx types.Tx) (*coretypes.ResultBroadcastTx, error) {
	// TODO 参数无法传进来
	//tx = &types.Tx([]byte("adf==1234"))
	err := env.Mempool.CheckTx(tx, meml.TxInfo{})

	if err != nil {
		return nil, err
	}
	return &coretypes.ResultBroadcastTx{Hash: tx.Hash()}, nil
}

func BroadcastTxTest(ctx *rpctypes.Context) (*coretypes.ResultBroadcastTx, error) {
	tx := types.Tx([]byte("adf==1234"))
	err := env.Mempool.CheckTx(tx, meml.TxInfo{})

	if err != nil {
		return nil, err
	}
	return &coretypes.ResultBroadcastTx{Hash: tx.Hash()}, nil
}

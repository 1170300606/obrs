package rpc

import rpc "github.com/tendermint/tendermint/rpc/jsonrpc/server"

var Routes = map[string]*rpc.RPCFunc{
	"broadcast_tx":      rpc.NewRPCFunc(BroadcastTxAsync, "tx"),
	"broadcast_tx_test": rpc.NewRPCFunc(BroadcastTxTest, ""),
}

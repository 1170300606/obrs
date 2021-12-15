package rpc

import rpc "github.com/tendermint/tendermint/rpc/jsonrpc/server"

var Routes = map[string]*rpc.RPCFunc{
	"broadcast_tx": rpc.NewRPCFunc(BroadcastTxAsync, "tx"),

	"block_tree":  rpc.NewRPCFunc(BlockTree, ""),
	"performance": rpc.NewRPCFunc(performance, "start,end"),

	"init_account": rpc.NewRPCFunc(InitSmallBankAccount, "name,saving,checking"),

	"metrics": rpc.NewRPCFunc(JSONMetrics, "label"),

	"net_info": rpc.NewRPCFunc(NetInfo, ""),
}

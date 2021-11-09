package rpc

import (
	"chainbft_demo/consensus"
	"chainbft_demo/mempool"
	"chainbft_demo/state"
)

var (
	env *Environment
)

func SetEnvironment(e *Environment) {
	env = e
}

type Environment struct {
	Mempool   mempool.Mempool
	Consensus *consensus.ConsensusState
	Store     state.Store
}

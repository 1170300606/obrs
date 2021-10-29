package rpc

import (
	"chainbft_demo/consensus"
	"chainbft_demo/libs/metric"
	"chainbft_demo/mempool"
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

	MetricSet *metric.MetricSet
}

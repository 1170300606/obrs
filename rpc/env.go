package rpc

import (
	"chainbft_demo/consensus"
	"chainbft_demo/libs/metric"
	"chainbft_demo/mempool"
	"chainbft_demo/state"
	jsoniter "github.com/json-iterator/go"
)

var (
	env  *Environment
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func SetEnvironment(e *Environment) {
	env = e
}

type Environment struct {
	Mempool   mempool.Mempool
	Consensus *consensus.ConsensusState
	Store     state.Store

	MetricSet *metric.MetricSet
}

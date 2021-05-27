package rpc

import "chainbft_demo/mempool"

var (
	env *Environment
)

func SetEnvironment(e *Environment) {
	env = e
}

type Environment struct {
	Mempool mempool.Mempool
}

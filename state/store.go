package state

import "chainbft_demo/types"

// 数据持久化接口
// v0.1的版本里不考虑实现
type Store interface {
	CommitBlock(State, *types.Block) ([]byte, error)

	SaveState(State) error
}

package store

import (
	"chainbft_demo/state"
	"chainbft_demo/types"
)

func NewMockStore() *MockStore {
	return &MockStore{}
}

type MockStore struct {
}

func (mock *MockStore) SaveState(state.State) error {
	return nil
}

func (mock *MockStore) CommitBlock(s state.State, block *types.Block) ([]byte, error) {
	return nil, nil
}

package store

import (
	"chainbft_demo/state"
	"chainbft_demo/types"
	tmdb "github.com/tendermint/tm-db"
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

func (mock *MockStore) GetDB() tmdb.DB {
	panic("implement me")
}

func (mock *MockStore) InitAccount(s string, i int, i2 int) error {
	panic("implement me")
}

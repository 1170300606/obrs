package mock

import (
	mempl "chainbft_demo/mempool"
	"chainbft_demo/types"
	"github.com/tendermint/tendermint/libs/clist"
)

// Mempool is an empty implementation of a Mempool, useful for testing.
type Mempool struct{}

var _ mempl.Mempool = Mempool{}

func (Mempool) Lock()     {}
func (Mempool) Unlock()   {}
func (Mempool) Size() int { return 0 }
func (Mempool) CheckTx(_ types.Tx, _ mempl.TxInfo) error {
	return nil
}
func (Mempool) ReapTxs(_ int64) types.Txs  { return types.Txs{} }
func (Mempool) ReapMaxTxs(_ int) types.Txs { return types.Txs{} }
func (Mempool) Update(
	_ int64,
	_ types.Txs,
) error {
	return nil
}
func (Mempool) Flush()                        {}
func (Mempool) TxsAvailable() <-chan struct{} { return make(chan struct{}) }
func (Mempool) TxsBytes() int64               { return 0 }

func (Mempool) TxsFront() *clist.CElement    { return nil }
func (Mempool) TxsWaitChan() <-chan struct{} { return nil }

func (Mempool) InitWAL() error { return nil }
func (Mempool) CloseWAL()      {}

package mempool

import (
	"chainbft_demo/types"
	"crypto/sha256"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/clist"
	"github.com/tendermint/tendermint/libs/log"
	"sync"
	"sync/atomic"
)

const (
	TxKeySize = 32
)

func NewListMempool(config *cfg.MempoolConfig, height int64, options ...ListMempoolOption) *ListMempool {
	mem := &ListMempool{
		height: height,
		config: config,
		txs:    clist.New(),
	}

	mem.cache = nopTxCache{}

	mem.txsAvailable = make(chan struct{}, 1)

	for _, option := range options {
		option(mem)
	}

	return mem
}

type ListMempool struct {
	// Atomic integers
	height   int64 // the last block Update()'d to
	txsBytes int64 // total size of mempool, in bytes

	txsAvailable chan struct{} // fires once for each height, when the mempool is not empty

	config *cfg.MempoolConfig

	updateMtx sync.RWMutex
	preCheck  PreCheckFunc

	txs    *clist.CList
	txsMap sync.Map

	// Keep a cache of already-seen txs.
	// This reduces the pressure on the proxyApp.
	cache txCache

	logger log.Logger
}

type ListMempoolOption func(memppol *ListMempool)

func SetPreCheck(precheck PreCheckFunc) ListMempoolOption {
	return func(mem *ListMempool) {
		mem.preCheck = precheck
	}
}

func (mem *ListMempool) SetLogger(logger log.Logger) {
	mem.logger = logger
}

func (mem *ListMempool) CheckTx(tx types.Tx, txinfo TxInfo) error {
	//TODO check tx

	// 先判断tx是否已经
	if _, ok := mem.txsMap.Load(TxKey(tx)); ok {
		return ErrTxInMap
	}

	memTx := &mempoolTx{
		height: mem.height,
		tx:     tx,
	}
	memTx.senders.Store(txinfo.SenderID, struct{}{})

	mem.logger.Info("added tx", "tx", tx, "txinfo", txinfo)
	mem.addTx(memTx)

	return nil
}

func (mem *ListMempool) ReapTxs(maxBytes int64) types.Txs {
	return nil
}

// Lock 锁定mempool的updateMtx读写锁的写锁
func (mem *ListMempool) Lock() {
	mem.updateMtx.Lock()
}

// UnLock 释放mempool的updateMtx读写锁的写锁
func (mem *ListMempool) UnLock() {
	mem.updateMtx.Unlock()
}

func (mem *ListMempool) Update(i int64, txs types.Txs) error {
	return nil
}

func (mem *ListMempool) Flush() {
}

func (mem *ListMempool) Size() int {
	return mem.txs.Len()
}

func (mem *ListMempool) TxsBytes() int64 {
	return atomic.LoadInt64(&mem.txsBytes)
}

// addTx 将tx加入到mempool的双向链表；
// 并且更新快速查询表txMap和mempool的tx总大小
func (mem *ListMempool) addTx(memTx *mempoolTx) {
	e := mem.txs.PushBack(memTx)
	mem.txsMap.Store(TxKey(memTx.tx), e)
	atomic.AddInt64(&mem.txsBytes, int64(len(memTx.tx)))
}

func (mem *ListMempool) TxsWaitChan() <-chan struct{} {
	return mem.txs.WaitChan()
}

func (mem *ListMempool) TxsFront() *clist.CElement {
	return mem.txs.Front()
}

// ------------------------------

type txCache interface {
	Reset()
	Push(tx types.Tx) bool
	Remove(tx types.Tx)
}

type nopTxCache struct {
}

func (cache nopTxCache) Reset() {
	return
}
func (cache nopTxCache) Push(tx types.Tx) bool {
	return true
}
func (cache nopTxCache) Remove(tx types.Tx) {
	return
}

type mempoolTx struct {
	height int64

	tx      types.Tx
	senders sync.Map
}

// Height returns the height for this transaction
func (memTx *mempoolTx) Height() int64 {
	return atomic.LoadInt64(&memTx.height)
}

// ------------------------------
// TxKey is the fixed length array hash used as the key in maps.
func TxKey(tx types.Tx) [TxKeySize]byte {
	return sha256.Sum256(tx)
}

func txID(tx []byte) []byte {
	return nil
}

package mempool

import (
	"chainbft_demo/types"
	"crypto/sha256"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/clist"
	"github.com/tendermint/tendermint/libs/log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	TxKeySize = 32
)

func NewListMempool(config *cfg.MempoolConfig, options ...ListMempoolOption) *ListMempool {
	mem := &ListMempool{
		slot:   types.LtimeZero,
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
	slot     types.LTime // the last block Update()'d to
	txsBytes int64       // total size of mempool, in bytes

	txsAvailable chan struct{} // fires once for each height, when the mempool is not empty

	config *cfg.MempoolConfig

	updateMtx sync.RWMutex
	preCheck  PreCheckFunc

	txs    *clist.CList
	txsMap sync.Map // Txkey(tx) => clist.CElement

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
	txSize := int(tx.ComputeSize())
	if err := mem.isFull(txSize); err != nil {
		return err
	}

	if txSize > mem.config.MaxTxBytes {
		return ErrTxTooLarge{
			max:    mem.config.MaxTxBytes,
			actual: txSize,
		}
	}

	// 先判断tx是否已经存在
	if !mem.cache.Push(tx) {
		if e, ok := mem.txsMap.Load(TxKey(tx)); ok {
			// 已经收到过tx交易，将tx对应的sender标志位置为true
			memTx := e.(*clist.CElement).Value.(*mempoolTx)
			memTx.senders.LoadOrStore(txinfo.SenderID, true)
		}

		return ErrTxInCache
	}

	memTx := &mempoolTx{
		slot: mem.slot,
		tx:   tx,
	}
	memTx.senders.Store(txinfo.SenderID, struct{}{})

	mem.addTx(memTx)
	mem.logger.Debug("added tx", "tx", tx, "txinfo", txinfo, "memLen", mem.txs.Len())

	return nil
}

// 协程安全
func (mem *ListMempool) ReapTxs(maxBytes int64) types.Txs {
	mem.updateMtx.RLock()
	defer mem.updateMtx.RUnlock()

	txs := make([]types.Tx, 0, mem.txs.Len())

	for e := mem.txs.Front(); e != nil; e = e.Next() {
		memTx := e.Value.(*mempoolTx)

		// TODO 如何计算txs的bytes，计算编码后的bytes大小还是前的
		dataSize := types.CaputeSizeForTxs(append(txs, memTx.tx))

		if maxBytes > -1 && dataSize > maxBytes {
			return txs
		}
		memTx.tx.MarkTime(types.MempoolReap, time.Now().UnixNano())
		txs = append(txs, memTx.tx)
	}

	return txs
}

// Safe for concurrent use by multiple goroutines.
func (mem *ListMempool) ReapMaxTxs(max int) types.Txs {
	mem.updateMtx.RLock()
	defer mem.updateMtx.RUnlock()

	if max < 0 {
		max = mem.txs.Len()
	}

	txs := make([]types.Tx, 0, func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}(max, mem.Size()))
	for e := mem.txs.Front(); e != nil && len(txs) <= max; e = e.Next() {
		memTx := e.Value.(*mempoolTx)
		txs = append(txs, memTx.tx)
		memTx.tx.MarkTime(types.MempoolReap, time.Now().UnixNano())
	}
	mem.logger.Debug("reapped all txs", "size", len(txs))
	return txs
}

// Lock 锁定mempool的updateMtx读写锁的写锁
func (mem *ListMempool) Lock() {
	mem.updateMtx.Lock()
}

// UnLock 释放mempool的updateMtx读写锁的写锁
func (mem *ListMempool) Unlock() {
	mem.updateMtx.Unlock()
}

// Caller负责加锁
func (mem *ListMempool) Update(slot types.LTime, toRemoveTxs types.Txs) error {
	for _, tx := range toRemoveTxs {
		// 将提交的交易添加到cache中
		mem.cache.Push(tx)

		if e, ok := mem.txsMap.Load(TxKey(tx)); ok {
			mem.removeTx(tx, e.(*clist.CElement), false)
		}
	}

	mem.slot = slot
	return nil
}

// TODO toLockTxs变更状态
// Caller负责加锁
func (mem *ListMempool) LockTxs(_ types.Txs) error {
	return nil
}

func (mem *ListMempool) Flush() {
	mem.updateMtx.RLock()
	defer mem.updateMtx.RUnlock()

	_ = atomic.SwapInt64(&mem.txsBytes, 0)
	mem.cache.Reset()

	// 不调用mem.removeTx， 效率太差
	for e := mem.txs.Front(); e != nil; e = e.Next() {
		mem.txs.Remove(e)
		e.DetachPrev()
	}

	mem.txsMap.Range(func(key, _ interface{}) bool {
		mem.txsMap.Delete(key)
		return true
	})
}

func (mem *ListMempool) Size() int {
	return mem.txs.Len()
}

func (mem *ListMempool) TxsBytes() int64 {
	return atomic.LoadInt64(&mem.txsBytes)
}

func (mem *ListMempool) TxsWaitChan() <-chan struct{} {
	return mem.txs.WaitChan()
}

func (mem *ListMempool) TxsFront() *clist.CElement {
	return mem.txs.Front()
}

// addTx 将tx加入到mempool的双向链表；
// 并且更新快速查询表txMap和mempool的tx总大小
func (mem *ListMempool) addTx(memTx *mempoolTx) {
	e := mem.txs.PushBack(memTx)
	memTx.tx.MarkTime(types.MempoolAdd, time.Now().UnixNano())
	mem.txsMap.Store(TxKey(memTx.tx), e)
	atomic.AddInt64(&mem.txsBytes, memTx.tx.ComputeSize())
}

func (mem *ListMempool) removeTx(tx types.Tx, e *clist.CElement, removeFromCache bool) {
	mem.txs.Remove(e)
	e.DetachPrev()
	mem.txsMap.Delete(TxKey(tx))
	atomic.AddInt64(&mem.txsBytes, int64(-tx.ComputeSize()))
	if removeFromCache {
		mem.cache.Remove(tx)
	}
}

func (mem *ListMempool) isFull(txSize int) error {
	memSize := mem.Size()
	txsBytes := mem.TxsBytes()
	if memSize >= mem.config.Size || txsBytes+int64(txSize) > mem.config.MaxTxsBytes {
		return ErrMempoolIsFull{
			numTxs:      memSize,
			maxTxs:      mem.config.Size,
			txsBytes:    txsBytes,
			maxTxsBytes: mem.config.MaxTxsBytes,
		}
	}
	return nil
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
func (cache nopTxCache) Push(_ types.Tx) bool {
	return true
}
func (cache nopTxCache) Remove(_ types.Tx) {
	return
}

type mempoolTx struct {
	slot types.LTime

	tx      types.Tx
	senders sync.Map
}

// Height returns the height for this transaction
func (memTx *mempoolTx) Height() types.LTime {
	// 是否会有bug 并发读写
	return memTx.slot
	//return atomic.LoadInt64(&memTx.slot)
}

// ------------------------------
// TxKey is the fixed length array hash used as the key in maps.
func TxKey(tx types.Tx) [TxKeySize]byte {
	return sha256.Sum256(tx.Hash())
}

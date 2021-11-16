package mempool

import (
	jsoniter "github.com/json-iterator/go"
	"sync"
)

func newMemMetric() *memMetric {
	return &memMetric{}
}

type memMetric struct {
	mtx           sync.RWMutex
	TxsNum        int   `json:"txs_num"`         //mempool中所有的交易总数
	PendingTxsNum int64 `json:"pending_txs_num"` // mempool中正在等待打包的交易总数
	LockedTxsNum  int64 `json:"locked_txs_nums"` // mempool中被锁定的交易总数
	TotalTxsNum   int64 `json:"total_txs_bytes"` // 目前mempool所有的交易的大小
}

func (mm *memMetric) JSONString() string {
	mm.mtx.RLock()
	defer mm.mtx.RUnlock()
	s, _ := jsoniter.MarshalToString(mm)
	return s
}

func (mm *memMetric) MarkTxsNum(txsnum int) {
	mm.mtx.Lock()
	defer mm.mtx.Unlock()
	mm.TxsNum = txsnum
}

func (mm *memMetric) MarkPendingTxsNum(pending_txs_num int64) {
	mm.mtx.Lock()
	defer mm.mtx.Unlock()
	mm.PendingTxsNum = pending_txs_num
}

func (mm *memMetric) MarkLockedTxsNum(locked_txs_num int64) {
	mm.mtx.Lock()
	defer mm.mtx.Unlock()
	mm.LockedTxsNum = locked_txs_num
}

func (mm *memMetric) MarkTotalTxsBytes(total_txs_bytes int64) {
	mm.mtx.Lock()
	defer mm.mtx.Unlock()
	mm.TotalTxsNum = total_txs_bytes
}

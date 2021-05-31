package mempool

import (
	"chainbft_demo/types"
	"github.com/tendermint/tendermint/p2p"
)

type Mempool interface {
	// CheckTx检验一个新交易是否合法，来决定能否将其加入到mempool中
	CheckTx(types.Tx, TxInfo) error

	// ReapTxs从mempool中打包交易，打包交易的大小小于maxBytes
	ReapTxs(maxBytes int64) types.Txs

	// ReapMaxTxs从mempool中取出caller指定数量的交易
	// 如果max是负数则表示取出mempool所有的交易
	ReapMaxTxs(max int) types.Txs

	// Lock locks the mempool，更新mempool前必须lock mempool
	Lock()

	// UnLock the Mempool
	Unlock()

	// Update committed交易从mempool中删去
	// NOTE: 该函数只能在block被提交后才能调用
	// NOTE: caller负责Lock/Unlock
	Update(int64, types.Txs) error

	// Flush将mempool中的所有交易和和cache清空
	Flush()

	// Size返回mempool中的交易条数
	Size() int

	// TxsBytes返回mempool所有交易的byte大小
	TxsBytes() int64
}

//--------------------------------------------------------------------------------

// PreCheckFunc is an optional filter executed before CheckTx and rejects
// transaction if false is returned. An example would be to ensure that a
// transaction doesn't exceeded the block size.
type PreCheckFunc func(types.Tx) error

// PostCheckFunc is an optional filter executed after CheckTx and rejects
// transaction if false is returned. An example would be to ensure a
// transaction doesn't require more gas than available for the block.
//type PostCheckFunc func(types.Tx, *abci.ResponseCheckTx) error

// TxInfo are parameters that get passed when attempting to add a tx to the
// mempool.
type TxInfo struct {
	// SenderID is the internal peer ID used in the mempool to identify the
	// sender, storing 2 bytes with each tx instead of 20 bytes for the p2p.ID.
	SenderID uint16
	// SenderP2PID is the actual p2p.ID of the sender, used e.g. for logging.
	SenderP2PID p2p.ID
}

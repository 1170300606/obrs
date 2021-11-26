package types

import (
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

const (
	MempoolReap   = "mempool_reap"
	MempoolAdd    = "mempool_add"
	ConsPrecommit = "consensus_precommit"
	ConsCommit    = "consensus_commit"
)

//type Tx interface {
//	Hash() []byte
//	ComputeSize() int64
//	CalculateTime()
//	MarkTime(string, int64)
//}

func NewTx() Tx {
	return Tx{}
}

// ===== normal tx =====
type NormalTx []byte

func (tx NormalTx) Hash() []byte {
	return tmhash.Sum(tx)
}

func (tx NormalTx) ComputeSize() int64 {
	return int64(len(tx))
}

func (tx NormalTx) CalculateTime() {
	return
}

func (tx NormalTx) MarkTime(string, int64) {
	return
}

// ===== tx array =====
type Txs []Tx

func CaputeSizeForTxs(txs []Tx) int64 {
	var dataSize int64

	for _, tx := range txs {
		dataSize += tx.ComputeSize()
	}

	return dataSize
}

func (txs Txs) Append(tx Txs) Txs {
	return append(txs, tx...)
}

// 返回交易形成的merkle tree的根value
func (txs Txs) Hash() []byte {
	txBzs := make([][]byte, len(txs))
	for i := 0; i < len(txs); i++ {
		txBzs[i] = txs[i].Hash()
	}
	return merkle.HashFromByteSlices(txBzs)
}

// ===== small bank Tx =====

type SmallBankTxType string

var (
	SBBalanceTx           = SmallBankTxType("balance")
	SBDepositCheckingTx   = SmallBankTxType("deposit_checking")
	SBTransactionSavingTx = SmallBankTxType("transaction_saving")
	SBAmalgamateTx        = SmallBankTxType("amalgamate")
	SBWriteCheckingTx     = SmallBankTxType("write_checking")
)

type Tx struct {
	TxType SmallBankTxType `json:"tx_type"`
	Args   []string        `json:"args"`

	// *timestamp$表示记录的是时间点，纳秒级，*Time$表示记录的是时间段，单位ms
	// 在调用CalculateTime前，*Time记录的同样也是时间点，CalculateTime函数会计算正确的时间
	TxSendTimestamp int64 `json:"tx_send_timestamp"` // 交易从客户端发出的时间
	//MempoolWaitTime        int64 `json:"mempool_wait_time"`        //mempool队列等待时间
	//ConsensusPrecommitTime int64 `json:"consensus_precommit_time"` // 第一段共识耗时
	//ConsensusCommitTime    int64 `json:"consensus_commit_time`     // 提交等待耗时
	//TxLatency              int64 `json:"latency"`                  // 交易延迟

	//MempoolAddTimestamp int64
	//ReapedTimestamp     int64
	//PrecommitTimestamp  int64
	//CommitTimestamp     int64
}

func (tx *Tx) Hash() []byte {
	h := tmhash.New()
	h.Write([]byte(tx.TxType))
	for _, arg := range tx.Args {
		h.Write([]byte(arg))
	}

	return h.Sum([]byte{})
}

func (tx *Tx) ComputeSize() int64 {
	s := 0
	s += len(tx.TxType)

	for _, arg := range tx.Args {
		s += len([]byte(arg))
	}

	s += 9 * 8
	return int64(s)
}

func (tx *Tx) CalculateTime() {
	//tx.MempoolWaitTime = tx.ReapedTimestamp - tx.MempoolAddTimestamp
	//tx.ConsensusPrecommitTime = tx.PrecommitTimestamp - tx.ReapedTimestamp
	//tx.ConsensusCommitTime = tx.CommitTimestamp - tx.ReapedTimestamp
	//tx.TxLatency = tx.CommitTimestamp - tx.TxSendTimestamp
}

func (tx *Tx) MarkTime(s string, t int64) {
	if s == MempoolAdd {
		//tx.MempoolAddTimestamp = t
	} else if s == MempoolReap {
		//tx.ReapedTimestamp = t
	} else if s == "consensus_precommit" {
		//tx.PrecommitTimestamp = t
	} else if s == "consensus_commit" {
		//tx.CommitTimestamp = t
	}
}

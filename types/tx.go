package types

import (
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

type Tx []byte

func (tx Tx) Hash() []byte {
	return tmhash.Sum(tx)
}

type Txs []Tx

func CaputeSizeForTxs(txs []Tx) int64 {
	var dataSize int64

	for _, tx := range txs {
		dataSize += int64(len(tx))
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

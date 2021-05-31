package types

type Tx []byte

func (tx *Tx) Hash() []byte {
	return []byte("test")
}

type Txs []Tx

func CaputeSizeForTxs(txs []Tx) int64 {
	var dataSize int64

	for _, tx := range txs {
		dataSize += int64(len(tx))
	}

	return dataSize
}

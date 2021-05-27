package types

type Tx []byte

func (tx *Tx) Hash() []byte {
	return []byte("test")
}

type Txs []Tx

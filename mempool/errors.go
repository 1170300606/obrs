package mempool

import "errors"

var (
	// ErrTxInCache is returned to the client if we saw tx earlier
	ErrTxInCache = errors.New("tx already exists in cache")
	ErrTxInMap   = errors.New("tx already exists in map")
)

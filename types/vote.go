package types

import (
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"time"
)

type Vote struct {
	Slot             int64            `json:"slot"`
	BlockHash        tmbytes.HexBytes `json:"block_hash"` // 如果该字段为空说明节点投反对票
	Timestamp        time.Time        `json:"timestamp"`
	ValidatorAddress Address          `json:"validator_address"`
	ValidatorIndex   int32            `json:"validator_index"`
	Signature        []byte           `json:"signature"`
}

package consensus

import (
	"chainbft_demo/types"
	"github.com/tendermint/tendermint/libs/log"
	"time"
)

// Slot提供逻辑时钟，内部完成节点之间的时钟同步
type Slot interface {

	// 获取当前的slot
	GetSlot() types.LTime

	// 获取超时channel
	Chan() <-chan timeoutInfo

	// 重置超时定时器
	Reset(duration time.Duration)

	SetLogger(logger log.Logger)
}

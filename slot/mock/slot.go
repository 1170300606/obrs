package mock

import (
	slot "chainbft_demo/slot"
	"chainbft_demo/types"
	"time"
)

// Slot is an empty implementation of a Slot, useful for testing.
type Slot struct{}

var _ slot.Slot = Slot{}

func (s Slot) GetSlot() types.LTime { return types.LtimeZero }

func (s Slot) GetTimeOutChan() <-chan struct{} { return nil }

func (s Slot) Reset(_ time.Duration) {}

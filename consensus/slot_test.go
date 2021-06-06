package consensus

import (
	"chainbft_demo/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	"testing"
	"time"
)

func getTestLogWithDebug() log.Logger {
	return log.NewFilter(log.TestingLogger(), log.AllowDebug())
}

func getTestLog() log.Logger {
	return log.TestingLogger()
}

// 测试slotClock启动后设置一个initialTimeout的默认超时时间，在此之前都不会收到任何事件
func TestSLotDefaultTimeout(t *testing.T) {
	sc := NewSlotClock(0)
	sc.SetLogger(getTestLogWithDebug())
	sc.OnStart()
	select {
	case <-sc.Chan():
		t.Error("有额外的超时事件")
	case <-time.NewTimer(10 * time.Second).C:
		break
	}

}

// 测试多次Reset，clock能否正确更新slot值
func TestRNormalCase(t *testing.T) {
	initialTime := types.LTime(100)
	sc := NewSlotClock(initialTime)
	sc.SetLogger(getTestLogWithDebug())
	sc.OnStart()

	for i := 0; i < 10; i++ {

		after := 3 * time.Second
		resetNow := time.Now()
		sc.ResetClock(after)

		select {
		case <-sc.Chan():
			break
		case <-time.NewTimer(4 * time.Second).C:
			t.Error("超时事件没有正确触发")
		}
		d := sc.GetLastUptTime().Sub(resetNow)

		assert.Equal(t, initialTime.Update(1), sc.GetSlot(), "login time更新错误")
		assert.Greater(t, d, after, "slotClock超时事件没有按照预设时间触发")
		assert.Equal(t, after, sc.GetLastDuration(), "duration更新错误")
		initialTime = initialTime.Update(1)
	}
}

// 测试Reset的正确性 - 首先设置一个长的超时事件，然后用一个小的时间间隔来reset
func TestResetClock(t *testing.T) {
	initialTime := types.LTime(100)
	sc := NewSlotClock(initialTime)
	sc.SetLogger(getTestLog())

	longDuration := 10 * time.Second
	shortDuration := 5 * time.Second
	sc.OnStart()

	sc.ResetClock(longDuration)

	time.Sleep(3 * time.Second)
	resetNow := time.Now()
	sc.ResetClock(shortDuration)

	select {
	case <-sc.Chan():
		d := sc.GetLastUptTime().Sub(resetNow)
		assert.Greater(t, d, shortDuration, "slotClock超时事件没有按照预设时间触发")
		assert.Equal(t, shortDuration, sc.GetLastDuration(), "duration更新错误")
		break
	case <-time.NewTimer(longDuration).C:
		t.Error("超时事件一直没有触发")
	}
}

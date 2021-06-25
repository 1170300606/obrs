package consensus

import (
	cstypes "chainbft_demo/consensus/types"
	"chainbft_demo/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	"time"
)

// Slot提供逻辑时钟，内部完成节点之间的时钟同步
// consensusState不负责SlotClock的启动
type SlotClock interface {
	service.Service
	// 获取当前的slot
	GetSlot() types.LTime

	// 返回上一次设置的超时时间
	GetLastDuration() time.Duration

	// 返回上一次超时事件触发的时间点
	GetLastUptTime() time.Time

	// 获取超时channel
	Chan() <-chan timeoutInfo

	// 重置超时定时器
	ResetClock(after time.Duration)

	SetLogger(logger log.Logger)
}

func NewSlotClock(initSlot types.LTime) SlotClock {
	sc := &slotClock{
		curslot:      initSlot,
		lastUptTime:  time.Now(),
		lastDuration: 100 * time.Second,
		t:            time.NewTimer(100 * time.Second),
		tockChan:     make(chan timeoutInfo),
		timerChan:    make(chan time.Time),
	}

	sc.BaseService = *service.NewBaseService(nil, "SlotClock", sc)

	return sc
}

type slotClock struct {
	service.BaseService

	curslot      types.LTime      // 当前的slot
	lastUptTime  time.Time        // 上一次更新slot的Unix时间戳
	lastDuration time.Duration    // 上一次启动的时间间隔
	t            *time.Timer      //内部定时器
	timerChan    chan time.Time   // time的chan 二转
	tockChan     chan timeoutInfo // 外部订阅超时chan

}

func (sc *slotClock) OnStart() error {
	sc.Logger.Info("slot clock starts.")
	go func() {
		for {
			select {
			case <-sc.Quit():
				// consensus通过停止baseService来停止slotClock
				return
			case now := <-sc.t.C:
				go func() { sc.timerChan <- now }()
			case now := <-sc.timerChan:
				// 超时事件发生，先更新内部的logic time，在往上返回新的时间
				sc.Logger.Debug("Time out", "curslot", sc.curslot, "nextslot", sc.curslot.Update(1), "duration", sc.lastDuration)
				sc.lastUptTime = now
				sc.curslot = sc.curslot.Update(1)
				ti := timeoutInfo{
					Duration: sc.lastDuration,
					Slot:     sc.curslot,
					Step:     cstypes.RoundStepSlot,
				}
				go func() { sc.tockChan <- ti }()
			}
		}
	}()
	return nil
}

func (sc *slotClock) Stop() error {
	if sc.t != nil {
		sc.t.Stop()
	}
	if err := sc.BaseService.Stop(); err != nil {
		sc.Logger.Error("failed trying to stop eventSwitch", "error", err)
	}

	return sc.BaseService.Stop()
}

func (s *slotClock) GetSlot() types.LTime {
	return s.curslot
}

func (sc *slotClock) GetLastUptTime() time.Time {
	return sc.lastUptTime
}

func (sc *slotClock) GetLastDuration() time.Duration {
	return sc.lastDuration
}

func (s *slotClock) Chan() <-chan timeoutInfo {
	return s.tockChan
}

func (s *slotClock) ResetClock(after time.Duration) {
	s.Logger.Debug("reset clock", "duration", after)
	s.lastDuration = after
	if !s.t.Stop() {
		select {
		case <-s.t.C:
		default:
		}
	}
	s.t.Reset(after)
}

func (s *slotClock) SetLogger(logger log.Logger) {
	s.Logger = logger
}

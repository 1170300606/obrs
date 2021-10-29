package metric

import (
	"errors"
	"sync"
)

var (
	ErrMetricLabelExist = errors.New("metric label already exist")
)

func NewMetricSet() *MetricSet {
	return &MetricSet{
		metrics: make(map[string]MetricItem),
	}
}

type MetricSet struct {
	mtx     sync.RWMutex
	metrics map[string]MetricItem
}

// SetMetrics - 根据label设置对应的Metrics，如果有存在的label，则返回error
func (ms *MetricSet) SetMetrics(label string, item MetricItem) error {
	if ms.HasMetrics(label) {
		return ErrMetricLabelExist
	}

	ms.mtx.Lock()
	ms.metrics[label] = item
	ms.mtx.Unlock()
	return nil
}

func (ms *MetricSet) HasMetrics(label string) bool {
	ms.mtx.RLock()
	_, existed := ms.metrics[label]
	ms.mtx.RUnlock()
	return existed
}

func (ms *MetricSet) GetMetrics(label string) MetricItem {
	if !ms.HasMetrics(label) {
		return nil
	}

	ms.mtx.RLock()
	defer ms.mtx.RUnlock()

	metric, _ := ms.metrics[label]
	return metric
}

func (ms *MetricSet) GetAlllabels() []string {
	keys := make([]string, 0, len(ms.metrics))

	for k, _ := range ms.metrics {
		keys = append(keys, k)
	}

	return keys
}

func (ms *MetricSet) GetAllMetrics() []MetricItem {
	vals := make([]MetricItem, 0, len(ms.metrics))

	for _, v := range ms.metrics {
		vals = append(vals, v)
	}

	return vals
}

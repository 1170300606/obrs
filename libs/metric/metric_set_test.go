package metric

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func newTestMetric() *MetricSet {
	m := NewMetricSet()
	m.metrics["TEST"] = &mockMetricItem{name: "TEST"}
	return m
}

func TestMetricSet_HasMetrics(t *testing.T) {
	metric := newTestMetric()

	assert.True(t, metric.HasMetrics("TEST"), "should contain label(TEST)")
	assert.False(t, metric.HasMetrics("FTEST"), "shouldn't contain label(FTEST)")

}

func TestMetricSet_SetMetrics(t *testing.T) {
	metric := newTestMetric()

	mockItem := &mockMetricItem{name: "TEST"}
	assert.NotNil(t, metric.SetMetrics("TEST", mockItem), "label(TEST)不应该设置成功")

	assert.Nil(t, metric.SetMetrics("TEST1", mockItem), "label(TEST1)应该设置成功")

	assert.True(t, metric.HasMetrics("TEST"), "should contain label(TEST)")
	assert.True(t, metric.HasMetrics("TEST1"), "should contain label(TEST1)")

}

func TestMetricSet_GetAlllabels(t *testing.T) {
	metric := newTestMetric()

	labels := metric.GetAlllabels()

	assert.Equal(t, 1, len(labels), "len(labels) == 1")
	assert.Equal(t, "TEST", labels[0], "labels[0] ==\"TEST\"")
}

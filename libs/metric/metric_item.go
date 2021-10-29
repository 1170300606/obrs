package metric

// MetricItem - 一个独立的metric模块对应一个MetricItem
// 实现时要使用
type MetricItem interface {
	JSONString() string
}

type mockMetricItem struct {
	name string
}

func (mock *mockMetricItem) JSONString() string {
	return mock.name
}

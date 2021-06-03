package types

type LTime int64

const (
	LtimeZero = LTime(0)
)

func (t LTime) Update(delta int) LTime {
	cur := int64(t)
	return LTime(cur + int64(delta))
}

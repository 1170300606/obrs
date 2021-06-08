package types

import "strconv"

type LTime int64

const (
	LtimeZero = LTime(0)
)

func (t LTime) Update(delta int) LTime {
	cur := int64(t)
	return LTime(cur + int64(delta))
}

func (t LTime) Hash() []byte {
	return strconv.AppendInt([]byte{}, int64(t), 10)
}

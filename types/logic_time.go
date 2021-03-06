package types

import "strconv"

// LTime 用来表示系统的逻辑时钟
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

func (t LTime) Equal(other LTime) bool {
	if int64(t) == int64(other) {
		return true
	}
	return false
}

func (t LTime) Greater(other LTime) bool {
	curT := int64(t)
	otherT := int64(other)
	if curT >= otherT {
		return true
	}
	return false
}

// (t-1) % val
func (t LTime) Mod(val int) int {
	return int(int64(t) % int64(val))
}

// return x - y
func (x LTime) Sub(y LTime) int {
	return int(x - y)
}

func (x LTime) Int64() int64 {
	return int64(x)
}

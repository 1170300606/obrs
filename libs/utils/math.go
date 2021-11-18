package utils

import (
	"sort"
)

// TODO

func Max(data ...float64) float64 {
	if len(data) == 0 {
		return -1.0
	}

	res := data[0]
	for _, datum := range data {
		if datum > res {
			res = datum
		}
	}
	return res
}

func Min(data ...float64) float64 {
	if len(data) == 0 {
		return -1.0
	}

	res := data[0]
	for _, datum := range data {
		if datum < res {
			res = datum
		}
	}
	return res
}

func Mean(data ...float64) float64 {
	if len(data) == 0 {
		return -1.0
	}

	sort.Float64s(data)
	var res float64
	if len(data)%2 == 1 {
		res = data[len(data)/2]
	} else {
		mid := len(data) / 2
		res = (data[mid] + data[mid+1]) / 2
	}
	return res
}

func Avg(data ...float64) float64 {
	if len(data) == 0 {
		return -1.0
	}

	res := 0.0
	for _, datum := range data {
		res += datum
	}

	return res / float64(len(data))
}

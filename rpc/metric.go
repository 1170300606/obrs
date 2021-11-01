package rpc

import (
	"fmt"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

type ResultMetrics struct {
	Metrics map[string]string `json:"metrics"`
}

func JSONMetrics(ctx *rpctypes.Context, label string) (*ResultMetrics, error) {
	result := &ResultMetrics{Metrics: make(map[string]string)}

	var labels []string
	if label != "" {
		labels = []string{label}
	} else {
		labels = env.MetricSet.GetAlllabels()
	}

	fmt.Println(label)
	fmt.Println(labels)
	for _, l := range labels {
		item := env.MetricSet.GetMetrics(l)
		if item != nil {
			result.Metrics[l] = item.JSONString()
		}
	}

	return result, nil
}

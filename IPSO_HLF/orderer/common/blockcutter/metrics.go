/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockcutter

import "github.com/Yunpeng-J/HLF-2.2/common/metrics"

var (
	blockFillDuration = metrics.HistogramOpts{
		Namespace:    "blockcutter",
		Name:         "block_fill_duration",
		Help:         "The time from first transaction enqueing to the block being cut in seconds.",
		LabelNames:   []string{"channel"},
		StatsdFormat: "%{#fqname}.%{channel}",
	}
	blockScheduleDuration = metrics.HistogramOpts{
		Namespace:    "blockcutter",
		Name:         "block_schedule_duration",
		Help:         "The time for FVS.",
		LabelNames:   []string{"channel"},
		StatsdFormat: "%{#fqname}.%{channel}",
	}
)

type Metrics struct {
	BlockFillDuration     metrics.Histogram
	BlockScheduleDuration metrics.Histogram
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		BlockFillDuration:     p.NewHistogram(blockFillDuration),
		BlockScheduleDuration: p.NewHistogram(blockScheduleDuration),
	}
}

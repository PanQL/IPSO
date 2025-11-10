package infra

import "sync/atomic"

var (
	Metric = NewMetricInstance()
)

type MetricInstance struct {
	Abort          int32
	IntegrateAbort int32
}

func NewMetricInstance() *MetricInstance {
	return &MetricInstance{
		Abort:          0,
		IntegrateAbort: 0,
	}
}

func (m *MetricInstance) AddAbort() {
	atomic.AddInt32(&m.Abort, 1)
}

func (m *MetricInstance) AddAbortBatch(batchSize int32) {
	if batchSize <= 0 {
		return
	}
	atomic.AddInt32(&m.Abort, batchSize)
}

func (m *MetricInstance) AddIntegrateAbort() {
	atomic.AddInt32(&m.IntegrateAbort, 1)
}

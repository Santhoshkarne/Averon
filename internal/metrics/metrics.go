package metrics

import (
	"sync/atomic"
)

type Metrics struct {
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
}

func (m *Metrics) IncTotal() {
	atomic.AddUint64(&m.TotalRequests, 1)
}
func (m *Metrics) IncSuccess() {
	atomic.AddUint64(&m.SuccessRequests, 1)
}
func (m *Metrics) IncFailed() {
	atomic.AddUint64(&m.FailedRequests, 1)
}

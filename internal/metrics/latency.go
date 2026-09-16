package metrics

import (
	"math"
	"sort"

	"sync"
	"time"
)

type LatencyTracker struct{
	mu sync.Mutex
	samples []float64
	maxSize int 
}

func NewLatencyTracker(maxSize int) *LatencyTracker{
	return &LatencyTracker{
		samples:make([]float64,0,maxSize),
		maxSize:maxSize,
	}
}


func (lt *LatencyTracker) Record(duration time.Duration){
	lt.mu.Lock()
	defer lt.mu.Unlock()
	ms := float64(duration.Nanoseconds()) / 1e6
	if len(lt.samples) >= lt.maxSize {
		// Evict oldest sample (FIFO)
		lt.samples = lt.samples[1:]
	}
	lt.samples = append(lt.samples, ms)
}

func (lt *LatencyTracker) Percentile(p float64) float64 {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	if len(lt.samples) == 0 {
		return 0
	}
	sorted := make([]float64, len(lt.samples))
	copy(sorted, lt.samples)
	sort.Float64s(sorted)
	rank := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))
	if lower == upper {
		return sorted[lower]
	}
	weight := rank - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func (lt *LatencyTracker) P50() float64 {
	return lt.Percentile(50)
}

func (lt *LatencyTracker) P95() float64 {
	return lt.Percentile(95)
}

func (lt *LatencyTracker) P99() float64 {
	return lt.Percentile(99)
}

func (lt *LatencyTracker) Count() int {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	return len(lt.samples)
}
var GlobalLatency = NewLatencyTracker(10000)

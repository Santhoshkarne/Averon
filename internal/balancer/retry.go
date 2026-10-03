package balancer


import (
	"math"
	"math/rand"
	"time"
)

type RetryConfig struct{
	MaxRetries int
	BaseDelay time.Duration
	MaxDelay time.Duration
}
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
	}
}

func(rc *RetryConfig) CalculateBackoff(attempt int) time.Duration{
	delay:= time.Duration(math.Pow(2,float64(attempt)))*rc.BaseDelay
	if delay>rc.MaxDelay{
	delay= rc.MaxDelay
	}
   jitter:=0.75+rand.Float64()*0.5
   delay=time.Duration(float64(delay)*jitter)
   return delay
}

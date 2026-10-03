package balancer

import (
	"sync"
	"time"

	"github.com/karnesanthosh/Averon/internal/logger"
)

type CircuitState int 

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string{
    switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

type CircuitBreaker struct {
 mu sync.Mutex
 state CircuitState
 failureCount int
 successCount int
 failureThreshold int
 successThreshold int
 timeout time.Duration
 lastFailed time.Time
 serverID string
 
}

func NewCircuitBreaker(serverID string, failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker{
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		serverID:         serverID,
	}
}

func( cb *CircuitBreaker) Allow() bool{
	cb.mu.Lock()
	
	defer cb.mu.Unlock()

	switch cb.state{
	case StateClosed:
		 return true

	case StateOpen:
		if time.Since(cb.lastFailed)>cb.timeout{
			cb.state=StateHalfOpen
			cb.successCount=0
			logger.Global.Info("circuit breaker state change", map[string]interface{}{
				"server":    cb.serverID,
				"from":      "OPEN",
				"to":        "HALF-OPEN",
				"reason":    "timeout expired, testing recovery",
			})
			return true
		}
		return false
	case StateHalfOpen:
		return true
	
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess(){
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state{
	case StateClosed:
		cb.failureCount=0
    case StateHalfOpen:
		cb.successCount++
		if cb.successCount>=cb.successThreshold{
			cb.state=StateClosed
			cb.failureCount = 0
			cb.successCount = 0
			logger.Global.Info("circuit breaker state change", map[string]interface{}{
				"server":    cb.serverID,
				"from":      "HALF-OPEN",
				"to":        "CLOSED",
				"reason":    "success threshold reached",
			})
		}
	}

}
	
func(cb *CircuitBreaker) RecordFailure(){
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state{
	case StateClosed:
		cb.failureCount++
		if cb.failureCount>=cb.failureThreshold{
			cb.state=StateOpen
			cb.lastFailed=time.Now()
			logger.Global.Info("circuit breaker state change", map[string]interface{}{
				"server":    cb.serverID,
				"from":      "CLOSED",
				"to":        "OPEN",
				"reason":    "failure threshold reached",
			})
		}
	case StateHalfOpen:
		cb.state=StateOpen
		cb.lastFailed=time.Now()
		cb.successCount = 0
		logger.Global.Info("circuit breaker state change", map[string]interface{}{
			"server":    cb.serverID,
			"from":      "HALF-OPEN",
			"to":        "OPEN",
			"reason":    "failure threshold reached",
		})
	}
}

func (cb *CircuitBreaker) GetState() CircuitState{
	cb.mu.Lock()
	
	defer cb.mu.Unlock()
	return cb.state
}
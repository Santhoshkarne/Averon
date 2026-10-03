package balancer

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
	"context"

	"github.com/karnesanthosh/Averon/internal/server"
	"github.com/karnesanthosh/Averon/internal/metrics"
	"github.com/karnesanthosh/Averon/internal/logger"
	
)

// ResponseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoadBalancer distributes incoming HTTP requests across backend servers
// using the Round Robin algorithm.
type LoadBalancer struct {
	servers []*server.Server
	Strategy Strategy
	current uint64 // atomic counter for round-robin index
	breakers map[string]*CircuitBreaker
	retryConfig *RetryConfig
	requestTimeout time.Duration
}

// NewLoadBalancer creates a new LoadBalancer with the given servers.
// It initializes the reverse proxy for each server.
func NewLoadBalancer(servers []*server.Server,strategyName string, cbFailures, cbSuccesses int, 
	cbTimeout time.Duration, retryCfg *RetryConfig, reqTimeout time.Duration) (*LoadBalancer, error) {
	for _, s := range servers {
		if err := s.SetupProxy(); err != nil {
			return nil, fmt.Errorf("failed to setup proxy for server %s: %w", s.ID, err)
		}
	}
 
	var strat Strategy
	switch strategyName{
	case "least_connections":
	    strat = NewLeastConnections(servers)
	case "ip_hash":
		strat = NewIPHash(servers)				
	case "weighted_round_robin":
		strat = NewWeightedRoundRobin(servers)
	default:
		strat = NewRoundRobinStrategy(servers)
	}
	breakers := make(map[string]*CircuitBreaker)
	if cbFailures > 0 {
		for _, s := range servers {
			breakers[s.ID] = NewCircuitBreaker(s.ID, cbFailures, cbSuccesses, cbTimeout)
			logger.Global.Info("circuit breaker created", map[string]interface{}{
				"server":             s.ID,
				"failure_threshold":  cbFailures,
				"success_threshold":  cbSuccesses,
				"timeout":            cbTimeout.String(),
			})
		}
	}

	if retryCfg == nil {
		retryCfg = DefaultRetryConfig()
	}
	// Default request timeout
	if reqTimeout == 0 {
		reqTimeout = 30 * time.Second
	}
	return &LoadBalancer{
		servers:        servers,
		Strategy:       strat,
		current:        0,
		breakers:       breakers,
		retryConfig:    retryCfg,
		requestTimeout: reqTimeout,
	}, nil
}

// ServeHTTP implements the http.Handler interface.
// This makes LoadBalancer usable directly as an HTTP server handler.
// For each incoming request, it picks the next server and forwards the request.
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Track total requests
	metrics.GlobalMetrics.IncTotal()

	// Try up to maxRetries + 1 times (1 original + N retries)
	maxAttempts := lb.retryConfig.MaxRetries + 1
	var lastErr string
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// If this is a retry, wait with exponential backoff
		if attempt > 0 {
			delay := lb.retryConfig.CalculateBackoff(attempt - 1)
			logger.Global.Warn("retrying request", map[string]interface{}{
				"attempt":  attempt,
				"max":      maxAttempts,
				"delay_ms": delay.Milliseconds(),
				"path":     r.URL.Path,
				"reason":   lastErr,
			})
			time.Sleep(delay)
		}

	srv := lb.Strategy.Next(r)
	if srv == nil {
		http.Error(w, "Service Unavailable: all backend servers are down", http.StatusServiceUnavailable)
		metrics.GlobalMetrics.IncFailed()
		return
	}
	if cb, exists := lb.breakers[srv.ID]; exists {
			if !cb.Allow() {
				// Circuit is OPEN — this server is considered broken.
				// Log and try the next attempt (which picks a different server)
				lastErr = fmt.Sprintf("circuit open for %s", srv.ID)
				logger.Global.Warn("circuit breaker blocked request", map[string]interface{}{
					"server": srv.ID,
					"state":  cb.GetState().String(),
				})
				continue // Try next attempt → strategy picks a different server
			}
		}

	logger.Global.Info("Forwarding request", map[string]interface{}{
		"method": r.Method,
		"path": r.URL.Path,
		"backend":srv.ID,
		"url":srv.URL,
		"attempt":attempt+1,

	})

	// Track request to this backend server
	atomic.AddUint64(&srv.Requests, 1)
	srv.IncrementConnections()

	ctx, cancel := context.WithTimeout(r.Context(), lb.requestTimeout)
		reqWithTimeout := r.WithContext(ctx)
	

	// Wrap the response writer to capture status code
	wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	srv.ReverseProxy.ServeHTTP(wrapped, reqWithTimeout)
		cancel() // Always cancel the context to free resources
		srv.DecrementConnections()

	// Delegate to the server's reverse proxy, which handles:
	// - Forwarding the request (method, headers, body) to the backend
	// - Streaming the response back to the client
	

	// Track success/failed based on status code
	// Evaluate the result
		if wrapped.statusCode >= 500 {
			// Server error (5xx) — record failure in circuit breaker
			if cb, exists := lb.breakers[srv.ID]; exists {
				cb.RecordFailure()
			}
			metrics.GlobalMetrics.IncFailed()
			lastErr = fmt.Sprintf("server %s returned %d", srv.ID, wrapped.statusCode)
			// Only retry on GET requests (safe to retry) or idempotent methods
			// POST/PUT/PATCH are NOT safe to retry (could create duplicates)
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				continue // Try again with a different server
			}
			// Non-idempotent method — don't retry, just return the error
			return
		}
		// Success! Record it in the circuit breaker
		if cb, exists := lb.breakers[srv.ID]; exists {
			cb.RecordSuccess()
		}
		if wrapped.statusCode >= 400 {
			metrics.GlobalMetrics.IncFailed()
		} else {
			metrics.GlobalMetrics.IncSuccess()
		}
		return // Request succeeded, we're done
	}
	// All retry attempts exhausted
	logger.Global.Error("all retry attempts failed", map[string]interface{}{
		"path":     r.URL.Path,
		"attempts": maxAttempts,
		"last_err": lastErr,
	})
	http.Error(w, "Service Unavailable: all retry attempts failed", http.StatusServiceUnavailable)
	metrics.GlobalMetrics.IncFailed()
}

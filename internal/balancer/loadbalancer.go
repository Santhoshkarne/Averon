package balancer

import (
	"fmt"
	"net/http"
	"sync/atomic"

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
}

// NewLoadBalancer creates a new LoadBalancer with the given servers.
// It initializes the reverse proxy for each server.
func NewLoadBalancer(servers []*server.Server,strategyName string) (*LoadBalancer, error) {
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

	return &LoadBalancer{
		servers: servers,
		Strategy: strat,
		current: 0,
	}, nil
}

// ServeHTTP implements the http.Handler interface.
// This makes LoadBalancer usable directly as an HTTP server handler.
// For each incoming request, it picks the next server and forwards the request.
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Track total requests
	metrics.GlobalMetrics.IncTotal()

	srv := lb.Strategy.Next(r)
	if srv == nil {
		http.Error(w, "Service Unavailable: all backend servers are down", http.StatusServiceUnavailable)
		metrics.GlobalMetrics.IncFailed()
		return
	}

	logger.Global.Info("Forwarding request", map[string]interface{}{
		"method": r.Method,
		"path": r.URL.Path,
		"backend":srv.ID,
		"url":srv.URL,	

	})

	// Track request to this backend server
	atomic.AddUint64(&srv.Requests, 1)
	srv.IncrementConnections()
	defer srv.DecrementConnections()
	

	// Wrap the response writer to capture status code
	wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// Delegate to the server's reverse proxy, which handles:
	// - Forwarding the request (method, headers, body) to the backend
	// - Streaming the response back to the client
	srv.ReverseProxy.ServeHTTP(wrapped, r)

	// Track success/failed based on status code
	if wrapped.statusCode >= 400 {
		metrics.GlobalMetrics.IncFailed()
	} else {
		metrics.GlobalMetrics.IncSuccess()
	}
}

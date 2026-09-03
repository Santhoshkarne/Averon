package server

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

// Server represents a backend server that can receive proxied requests.
type Server struct {
	ID           string                 `json:"id"`
	URL          string                 `json:"url"`
	ReverseProxy *httputil.ReverseProxy `json:"-"`

	// alive is accessed from multiple goroutines (health checker + request handler),
	// so we protect it with a mutex to avoid data races.
	alive bool
	mu    sync.RWMutex

	Requests uint64
	// ActiveConnections tracks in-flight requests for least-connections balancing.
	ActiveConnections int64
}

// IsAlive returns whether the server is currently healthy.
// Thread-safe: uses a read lock.
func (s *Server) IsAlive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.alive
}

// SetAlive updates the server's health status.
// Thread-safe: uses a write lock.
func (s *Server) SetAlive(alive bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alive = alive
}

// IncrementConnections atomically increments the active connection count.
func (s *Server) IncrementConnections() {
	atomic.AddInt64(&s.ActiveConnections, 1)
}

// DecrementConnections atomically decrements the active connection count.
func (s *Server) DecrementConnections() {
	atomic.AddInt64(&s.ActiveConnections, -1)
}

// GetConnections atomically reads the active connection count.
func (s *Server) GetConnections() int64 {
	return atomic.LoadInt64(&s.ActiveConnections)
}

// SetupProxy parses the server URL and creates a reverse proxy for it.
// This must be called after loading the server from config.
func (s *Server) SetupProxy() error {
	targetURL, err := url.Parse(s.URL)
	if err != nil {
		return err
	}

	s.ReverseProxy = httputil.NewSingleHostReverseProxy(targetURL)
	return nil
}

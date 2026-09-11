package balancer

import (
	"net/http"
	"sync/atomic"

	"github.com/karnesanthosh/Averon/internal/server"
)

type RoundRobinStrategy struct {
	servers []*server.Server
	current uint64
}

func NewRoundRobinStrategy(servers []*server.Server) *RoundRobinStrategy {
	return &RoundRobinStrategy{servers: servers}
}

func (s *RoundRobinStrategy) Next(req *http.Request) *server.Server {
	total := uint64(len(s.servers))
	if total == 0 {
		return nil
	}

	for i := uint64(0); i < total; i++ {
		next := atomic.AddUint64(&s.current, 1)
		idx := (next - 1) % total
		if s.servers[idx].IsAlive() {
			return s.servers[idx]
		}
	}
	return nil
}

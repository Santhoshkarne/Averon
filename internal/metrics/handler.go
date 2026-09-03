package metrics

import (
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/SaisrikarVollala/nebulagate/internal/server"
)

func Hanlder(w http.ResponseWriter, r *http.Request) {

	response := map[string]uint64{
		"total_requests":   atomic.LoadUint64(&GlobalMetrics.TotalRequests),
		"success_requests": atomic.LoadUint64(&GlobalMetrics.SuccessRequests),
		"failed_requests":  atomic.LoadUint64(&GlobalMetrics.FailedRequests),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// NewHandler creates a metrics handler with access to backend servers
func NewHandler(servers []*server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"total_requests":   atomic.LoadUint64(&GlobalMetrics.TotalRequests),
			"success_requests": atomic.LoadUint64(&GlobalMetrics.SuccessRequests),
			"failed_requests":  atomic.LoadUint64(&GlobalMetrics.FailedRequests),
			"backends":         make(map[string]uint64),
		}

		// Add per-backend request counts
		backends := response["backends"].(map[string]uint64)
		for _, srv := range servers {
			backends[srv.ID] = atomic.LoadUint64(&srv.Requests)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

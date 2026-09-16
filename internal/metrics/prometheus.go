package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/karnesanthosh/Averon/internal/server"
)

// PrometheusHandler returns an HTTP handler that exposes metrics in Prometheus format.
//
// Prometheus scrapes this endpoint every 15 seconds (configurable) and stores the data
// as time-series. You can then build Grafana dashboards with real-time graphs.
//
// Format reference: https://prometheus.io/docs/instrumenting/exposition_formats/
func PrometheusHandler(servers []*server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// --- Request counters ---
		fmt.Fprintf(w, "# HELP nebulagate_requests_total Total number of HTTP requests processed.\n")
		fmt.Fprintf(w, "# TYPE nebulagate_requests_total counter\n")
		fmt.Fprintf(w, "nebulagate_requests_total %d\n\n",
			atomic.LoadUint64(&GlobalMetrics.TotalRequests))

		fmt.Fprintf(w, "# HELP nebulagate_requests_success_total Total successful requests (2xx/3xx).\n")
		fmt.Fprintf(w, "# TYPE nebulagate_requests_success_total counter\n")
		fmt.Fprintf(w, "nebulagate_requests_success_total %d\n\n",
			atomic.LoadUint64(&GlobalMetrics.SuccessRequests))

		fmt.Fprintf(w, "# HELP nebulagate_requests_failed_total Total failed requests (4xx/5xx).\n")
		fmt.Fprintf(w, "# TYPE nebulagate_requests_failed_total counter\n")
		fmt.Fprintf(w, "nebulagate_requests_failed_total %d\n\n",
			atomic.LoadUint64(&GlobalMetrics.FailedRequests))

		// --- Latency percentiles (gauge) ---
		fmt.Fprintf(w, "# HELP nebulagate_request_duration_ms Request duration percentiles in milliseconds.\n")
		fmt.Fprintf(w, "# TYPE nebulagate_request_duration_ms gauge\n")
		fmt.Fprintf(w, "nebulagate_request_duration_ms{quantile=\"0.5\"} %.2f\n", GlobalLatency.P50())
		fmt.Fprintf(w, "nebulagate_request_duration_ms{quantile=\"0.95\"} %.2f\n", GlobalLatency.P95())
		fmt.Fprintf(w, "nebulagate_request_duration_ms{quantile=\"0.99\"} %.2f\n\n", GlobalLatency.P99())

		// --- Per-backend metrics ---
		fmt.Fprintf(w, "# HELP nebulagate_backend_requests_total Total requests forwarded to each backend.\n")
		fmt.Fprintf(w, "# TYPE nebulagate_backend_requests_total counter\n")
		for _, srv := range servers {
			fmt.Fprintf(w, "nebulagate_backend_requests_total{backend=\"%s\"} %d\n",
				srv.ID, atomic.LoadUint64(&srv.Requests))
		}
		fmt.Fprintf(w, "\n")

		fmt.Fprintf(w, "# HELP nebulagate_backend_active_connections Current in-flight requests per backend.\n")
		fmt.Fprintf(w, "# TYPE nebulagate_backend_active_connections gauge\n")
		for _, srv := range servers {
			fmt.Fprintf(w, "nebulagate_backend_active_connections{backend=\"%s\"} %d\n",
				srv.ID, srv.GetConnections())
		}
		fmt.Fprintf(w, "\n")

		fmt.Fprintf(w, "# HELP nebulagate_backend_alive Whether the backend is currently healthy (1=up, 0=down).\n")
		fmt.Fprintf(w, "# TYPE nebulagate_backend_alive gauge\n")
		for _, srv := range servers {
			alive := 0
			if srv.IsAlive() {
				alive = 1
			}
			fmt.Fprintf(w, "nebulagate_backend_alive{backend=\"%s\"} %d\n",
				srv.ID, alive)
		}
	}
}

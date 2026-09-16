package middleware

import (
	"net/http"
	"strings"
	"time"
	"github.com/karnesanthosh/Averon/internal/logger"
	"github.com/karnesanthosh/Averon/internal/metrics"
)

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}
func (w *statusResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}
func (w *statusResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = http.StatusOK
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

func Logging(next http.Handler) http.Handler{
	return http.HandlerFunc(func (w http.ResponseWriter,r *http.Request){
		start:= time.Now()

		wrapped:= &statusResponseWriter{
			ResponseWriter: w,
			statusCode: http.StatusOK,
		}
		next.ServeHTTP(wrapped,r)

		duration :=time.Since(start)
		durationMs := float64(duration.Milliseconds())

		metrics.GlobalLatency.Record(duration)

		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = strings.Split(forwarded, ",")[0]
		}
		requestID := GetRequestID(r.Context())

		logger.Global.Info("request completed", map[string]interface{}{
			"request_id":  requestID,
			"method":      r.Method,
			"path":        r.URL.Path,
			"query":       r.URL.RawQuery,
			"status":      wrapped.statusCode,
			"duration_ms": durationMs,
			"client_ip":   clientIP,
			"user_agent":  r.UserAgent(),
		})


		
			
	})
}




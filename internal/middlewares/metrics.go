package middlewares

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ocenb/marketplace/internal/metrics"
)

func MetricsMiddleware(metrics *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := newWrappedResponseWriter(w)

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()

			metrics.RequestsCounter.WithLabelValues(
				r.Method,
				r.URL.Path,
				strconv.Itoa(ww.statusCode),
			).Inc()

			metrics.ResponseTime.WithLabelValues(
				r.Method,
				r.URL.Path,
			).Observe(duration)
		})
	}
}

type wrappedResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newWrappedResponseWriter(w http.ResponseWriter) *wrappedResponseWriter {
	return &wrappedResponseWriter{w, http.StatusOK}
}

func (ww *wrappedResponseWriter) WriteHeader(code int) {
	ww.statusCode = code
	ww.ResponseWriter.WriteHeader(code)
}

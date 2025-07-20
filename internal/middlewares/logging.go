package middlewares

import (
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(statusCode int) {
	rec.statusCode = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

func LoggingMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			recorder := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			log.Info("Request", slog.String("method", r.Method), slog.String("path", r.URL.Path))

			h.ServeHTTP(recorder, r)

			log.Info("Response",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", recorder.statusCode),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

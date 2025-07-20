package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ocenb/marketplace/internal/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	httpServer *http.Server
	log        *slog.Logger
}

func NewServer(port string, log *slog.Logger) *Server {
	router := http.NewServeMux()

	router.Handle("/metrics", promhttp.Handler())

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: router,
		},
		log: log,
	}
}

func (s *Server) Start() {
	go func() {
		s.log.Info("Starting Prometheus metrics server", slog.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("Metrics server error", utils.ErrLog(err))
		}
	}()
}

func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("Stopping Prometheus metrics server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

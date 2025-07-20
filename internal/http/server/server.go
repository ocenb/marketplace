package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ocenb/marketplace/internal/config"
	"github.com/ocenb/marketplace/internal/metrics"
	"github.com/ocenb/marketplace/internal/middlewares"
	"github.com/ocenb/marketplace/internal/utils"
)

type HttpServer struct {
	log        *slog.Logger
	router     *chi.Mux
	httpServer *http.Server
	cfg        *config.Config
}

func NewHttpServer(log *slog.Logger, cfg *config.Config) *HttpServer {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middlewares.LoggingMiddleware(log))

	return &HttpServer{
		log:    log,
		router: router,
		cfg:    cfg,
	}
}

func (s *HttpServer) AddMetricsMiddleware(metricsInstance *metrics.Metrics) {
	s.router.Use(middlewares.MetricsMiddleware(metricsInstance))
}

func (s *HttpServer) Router() *chi.Mux {
	return s.router
}

func (s *HttpServer) Start() error {
	s.log.Info("Starting HTTP server", slog.String("port", s.cfg.Server.ServerPort))

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%s", s.cfg.Server.ServerPort),
		Handler:           s.router,
		ReadTimeout:       s.cfg.Server.ReadTimeout,
		WriteTimeout:      s.cfg.Server.WriteTimeout,
		IdleTimeout:       s.cfg.Server.IdleTimeout,
		ReadHeaderTimeout: s.cfg.Server.ReadHeaderTimeout,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("HTTP server error", utils.ErrLog(err))
		}
	}()

	return nil
}

func (s *HttpServer) Stop(ctx context.Context) error {
	s.log.Info("Stopping HTTP server")

	return s.httpServer.Shutdown(ctx)
}

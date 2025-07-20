package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	_ "github.com/ocenb/marketplace/docs"
	"github.com/ocenb/marketplace/internal/config"
	authhandler "github.com/ocenb/marketplace/internal/handlers/auth"
	listinghandler "github.com/ocenb/marketplace/internal/handlers/listing"
	"github.com/ocenb/marketplace/internal/http/server"
	"github.com/ocenb/marketplace/internal/logger"
	"github.com/ocenb/marketplace/internal/metrics"
	"github.com/ocenb/marketplace/internal/middlewares"
	authrepo "github.com/ocenb/marketplace/internal/repos/auth"
	listingrepo "github.com/ocenb/marketplace/internal/repos/listing"
	userrepo "github.com/ocenb/marketplace/internal/repos/user"
	authservice "github.com/ocenb/marketplace/internal/services/auth"
	listingservice "github.com/ocenb/marketplace/internal/services/listing"
	userservice "github.com/ocenb/marketplace/internal/services/user"
	"github.com/ocenb/marketplace/internal/storage/postgres"
	"github.com/ocenb/marketplace/internal/utils"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Marketplace API
// @version 1.0

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" + your JWT token in the input box below."
func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg)

	log.Info("Connecting to database",
		slog.String("host", cfg.Postgres.Host),
		slog.String("port", cfg.Postgres.Port),
		slog.String("database", cfg.Postgres.Name),
	)
	postgres, err := postgres.New(cfg)
	if err != nil {
		log.Error("Failed to connect to postgres", utils.ErrLog(err))
		os.Exit(1)
	}
	defer func() {
		log.Info("Closing database connection")
		err := postgres.Close()
		if err != nil {
			log.Error("Failed to close postgres connection", utils.ErrLog(err))
		}
	}()

	validator := validator.New()

	metricsInstance := metrics.NewMetrics("marketplace")
	metricsServer := metrics.NewServer(cfg.Server.MetricsPort, log)
	metricsServer.Start()

	authRepo := authrepo.New(postgres)
	userRepo := userrepo.New(postgres)
	listingRepo := listingrepo.New(postgres, log)

	userService := userservice.New(userRepo)
	authService := authservice.New(cfg, log, authRepo, userService)
	listingService := listingservice.New(listingRepo, metricsInstance)

	authHandler := authhandler.New(authService, log, validator)
	listingHandler := listinghandler.New(listingService, log, validator)

	httpServer := server.NewHttpServer(log, cfg)
	httpServer.AddMetricsMiddleware(metricsInstance)

	router := httpServer.Router()
	authRouter := router.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware(log, authService))
	})
	optionalAuthRouter := router.Group(func(r chi.Router) {
		r.Use(middlewares.OptionalAuthMiddleware(log, authService))
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Error("Failed to write health response", utils.ErrLog(err))
		}
	})
	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	authHandler.RegisterRoutes(router)
	listingHandler.RegisterRoutes(optionalAuthRouter, authRouter)

	go runTokenCleanup(authService, log)

	if err := httpServer.Start(); err != nil {
		log.Error("Failed to start HTTP server", utils.ErrLog(err))
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	log.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Stop(ctx); err != nil {
		log.Error("HTTP server shutdown error", utils.ErrLog(err))
	}

	if err := metricsServer.Stop(ctx); err != nil {
		log.Error("Metrics server shutdown error", utils.ErrLog(err))
	}

	log.Info("Server gracefully stopped")
}

func runTokenCleanup(authService authservice.AuthServiceInterface, log *slog.Logger) {
	log.Info("Token cleanup scheduled", slog.Duration("interval", 24*time.Hour))
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	authService.CleanupExpiredTokens()
	for range ticker.C {
		authService.CleanupExpiredTokens()
	}
}

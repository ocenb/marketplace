package logger

import (
	"io"
	"log/slog"
	"os"

	"github.com/ocenb/marketplace/internal/config"
)

type HandlerType string

const (
	TextHandler  HandlerType = "text"
	JSONHandler  HandlerType = "json"
	DefaultLevel slog.Level  = slog.LevelInfo
)

func New(cfg *config.Config) *slog.Logger {
	handlerType := cfg.Log.Handler
	logLevel := cfg.Log.Level

	level := slog.LevelInfo
	if logLevel >= int(slog.LevelDebug) && logLevel <= int(slog.LevelError) {
		level = slog.Level(logLevel)
	} else {
		slog.Error("Invalid log level, using default level Info")
	}

	opts := &slog.HandlerOptions{Level: level, AddSource: true}
	var logger *slog.Logger

	switch HandlerType(handlerType) {
	case TextHandler:
		logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	case JSONHandler:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	default:
		slog.Error("Invalid log handler type, using default TextHandler")
		logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	}

	logger = logger.With(slog.String("env", cfg.Environment))

	return logger
}

func NewForTest() *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelError + 1}
	handler := slog.NewTextHandler(io.Discard, opts)

	return slog.New(handler)
}

package utils

import (
	"context"
	"log/slog"
)

type UserIDKey struct{}

func ErrLog(err error) slog.Attr {
	if err == nil {
		return slog.Any("error", nil)
	}
	return slog.String("error", err.Error())
}

func OpLog(op string) slog.Attr {
	return slog.String("op", op)
}

func GetInfoFromContext(ctx context.Context, log *slog.Logger) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey{}).(int64)
	if !ok {
		log.Info("Failed to get user from context")
		return -1, false
	}
	return userID, true
}

package middlewares

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ocenb/marketplace/internal/services/auth"
	"github.com/ocenb/marketplace/internal/utils"
	"github.com/ocenb/marketplace/internal/utils/httputil"
)

func AuthMiddleware(log *slog.Logger, authService auth.AuthServiceInterface) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := validateToken(r, log, authService)
			if err != nil {
				httputil.UnauthorizedError(w, log, "unauthorized")
				return
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuthMiddleware(log *slog.Logger, authService auth.AuthServiceInterface) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := validateToken(r, log, authService)
			if err != nil {
				h.ServeHTTP(w, r)
				return
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func validateToken(r *http.Request, log *slog.Logger, authService auth.AuthServiceInterface) (context.Context, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Info("Authorization header is missing")
		return nil, errors.New("authorization header is missing")
	}

	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		log.Info("Invalid authorization header format")
		return nil, errors.New("invalid authorization header format")
	}

	token := tokenParts[1]
	userID, err := authService.ValidateToken(r.Context(), token)
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(r.Context(), utils.UserIDKey{}, userID)

	return ctx, nil
}

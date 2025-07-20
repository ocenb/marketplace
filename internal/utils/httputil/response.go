package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ocenb/marketplace/internal/utils"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, data any, statusCode int, log *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Error("Failed to encode and write JSON response",
				slog.Int("status", statusCode),
				utils.ErrLog(err),
			)
		} else {
			log.Debug("Successfully wrote JSON response",
				slog.Int("status", statusCode),
			)
		}
	} else {
		log.Debug("Wrote response with no body",
			slog.Int("status", statusCode),
		)
	}
}

var (
	InternalError = func(w http.ResponseWriter, log *slog.Logger) {
		WriteJSON(w, ErrorResponse{Message: "internal server error"}, http.StatusInternalServerError, log)
	}

	NotFoundError = func(w http.ResponseWriter, log *slog.Logger, msg string) {
		WriteJSON(w, ErrorResponse{Message: msg}, http.StatusNotFound, log)
	}

	UnauthorizedError = func(w http.ResponseWriter, log *slog.Logger, msg string) {
		WriteJSON(w, ErrorResponse{Message: msg}, http.StatusUnauthorized, log)
	}

	BadRequestError = func(w http.ResponseWriter, log *slog.Logger, msg string) {
		WriteJSON(w, ErrorResponse{Message: msg}, http.StatusBadRequest, log)
	}

	ConflictError = func(w http.ResponseWriter, log *slog.Logger, msg string) {
		WriteJSON(w, ErrorResponse{Message: msg}, http.StatusConflict, log)
	}

	ForbiddenError = func(w http.ResponseWriter, log *slog.Logger) {
		WriteJSON(w, ErrorResponse{Message: "forbidden"}, http.StatusForbidden, log)
	}
)

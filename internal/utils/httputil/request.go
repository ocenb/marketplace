package httputil

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ocenb/marketplace/internal/utils"
)

const maxImageSize = 5 * 1024 * 1024 // 5 MB

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func DecodeAndValidate(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
	validate *validator.Validate,
	log *slog.Logger,
) bool {
	err := json.NewDecoder(r.Body).Decode(dst)
	if err != nil {
		log.Warn("Failed to decode request payload", utils.ErrLog(err))
		BadRequestError(w, log, "Invalid request payload")
		return false
	}

	err = validate.Struct(dst)
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			log.Info("Validation failed", utils.ErrLog(err))
			BadRequestError(w, log, fmt.Sprintf("Validation failed: %s", validationErrors.Error()))
			return false
		}
		log.Error("Unexpected validation error type", utils.ErrLog(err))
		InternalError(w, log)
		return false
	}

	return true
}

func ValidateImage(log *slog.Logger, url string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to make request to %s: %w", url, err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Error("Failed to close response body", utils.ErrLog(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received incorrect status code: %d", resp.StatusCode)
	}

	sizeStr := resp.Header.Get("Content-Length")
	if sizeStr == "" {
		return fmt.Errorf("Content-Length header not found")
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid Content-Length header: %w", err)
	}

	if size > maxImageSize {
		return fmt.Errorf("image size (%d bytes) exceeds limit (%d bytes)", size, maxImageSize)
	}

	buffer := make([]byte, 512)
	n, err := io.ReadFull(resp.Body, buffer)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("image file is empty")
	}

	contentType := http.DetectContentType(buffer[:n])
	if contentType != "image/jpeg" && contentType != "image/png" {
		return fmt.Errorf("invalid file type: expected 'image/jpeg' or 'image/png', got '%s'", contentType)
	}

	return nil
}

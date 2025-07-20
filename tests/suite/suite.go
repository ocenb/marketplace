package suite

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

type Suite struct {
	*testing.T
	BaseURL string
	Client  *http.Client
	ctx     context.Context
}

func New(t *testing.T) *Suite {
	t.Helper()
	t.Parallel()

	baseURL := os.Getenv("TEST_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	t.Cleanup(func() {
		t.Helper()
		cancel()
	})

	s := &Suite{
		T:       t,
		BaseURL: baseURL,
		Client:  client,
		ctx:     ctx,
	}

	s.waitForService()

	return s
}

func (s *Suite) waitForService() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.Fatalf("Timeout waiting for service to start")
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/health", s.BaseURL), nil)
			if err != nil {
				continue
			}

			resp, err := s.Client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				err := resp.Body.Close()
				if err != nil {
					s.Errorf("Failed to close response body: %v", err)
				}
				return
			}
			if resp != nil {
				err := resp.Body.Close()
				if err != nil {
					s.Errorf("Failed to close response body: %v", err)
				}
			}
		}
	}
}

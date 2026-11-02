package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// runHealthcheck GETs url and returns nil only for a 200 response. It exists so the
// container image can ship without curl or wget: HEALTHCHECK runs `api -healthcheck`.
// The timeout is required so a hung server can never hang the probe.
func runHealthcheck(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("healthcheck request invalid: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck got status %d", resp.StatusCode)
	}
	return nil
}

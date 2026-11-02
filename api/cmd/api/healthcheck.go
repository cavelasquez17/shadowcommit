package main

import (
	"fmt"
	"net/http"
	"time"
)

// runHealthcheck GETs url and returns nil only for a 200 response. It exists so the
// container image can ship without curl or wget: HEALTHCHECK runs `api -healthcheck`.
// The timeout is required so a hung server can never hang the probe.
func runHealthcheck(url string, timeout time.Duration) error {
	resp, err := (&http.Client{Timeout: timeout}).Get(url)
	if err != nil {
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck got status %d", resp.StatusCode)
	}
	return nil
}

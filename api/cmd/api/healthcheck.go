package main

import (
	"fmt"
	"net/http"
	"time"
)

// runHealthcheck GETs url and returns nil only for a 200 response. It exists so the
// container image can ship without curl or wget: HEALTHCHECK runs `api -healthcheck`.
func runHealthcheck(client *http.Client, url string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck got status %d", resp.StatusCode)
	}
	return nil
}

func healthClient() *http.Client {
	return &http.Client{Timeout: 3 * time.Second}
}

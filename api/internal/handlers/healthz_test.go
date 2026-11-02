package handlers_test

import (
	"net/http"
	"testing"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
)

func TestHealthzReportsStatusAndVersion(t *testing.T) {
	t.Parallel()
	ts, _, _ := newTestServer(t, func(c *config.Config) { c.Version = "abc1234" })
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz: got %d, want 200", resp.StatusCode)
	}
	body := decodeJSON[map[string]string](t, resp)
	if body["status"] != "alive" || body["version"] != "abc1234" {
		t.Errorf("healthz = %v, want status=alive version=abc1234", body)
	}
}

func TestHealthzFallsBackToDevVersion(t *testing.T) {
	t.Parallel()
	ts, _, _ := newTestServer(t, nil)
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeJSON[map[string]string](t, resp)["version"]; got != "dev" {
		t.Errorf("version without stamping = %q, want dev", got)
	}
}

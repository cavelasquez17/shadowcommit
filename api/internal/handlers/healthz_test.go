package handlers_test

import (
	"net/http"
	"testing"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
)

func TestHealthzReportsStatusAndVersion(t *testing.T) {
	t.Parallel()
	ts, _, _ := newTestServer(t, func(c *config.Config) { c.Version = "abc1234" })
	resp := mustGet(t, ts.URL+"/healthz")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz: got %d, want 200", resp.StatusCode)
	}
	body := decodeJSON[map[string]string](t, resp)
	if body["status"] != "alive" || body["version"] != "abc1234" {
		t.Errorf("healthz = %v, want status=alive version=abc1234", body)
	}
}

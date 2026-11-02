package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/handlers"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
	"github.com/cavelasquez17/shadowcommit/api/internal/testutil"
)

// newTestServer boots a full API — real in-memory sqlite, real router, real middleware —
// behind an httptest.Server so tests drive it exactly like a browser or the SPA would.
func newTestServer(t *testing.T, mutate func(*config.Config)) (*httptest.Server, *gorm.DB) {
	t.Helper()
	cfg := &config.Config{
		DBDriver:        "sqlite",
		DSN:             testutil.MemoryDSN(t),
		JWTSecret:       "test-secret",
		AppBaseURL:      "http://localhost:8080",
		WebBaseURL:      "http://localhost:5173",
		CORSOrigins:     []string{"http://localhost:5173"},
		DevLoginEnabled: true,
	}
	if mutate != nil {
		mutate(cfg)
	}
	db, err := store.Open(cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	iss := auth.NewIssuer(cfg)
	api := handlers.New(db, cfg, iss, nil)
	t.Cleanup(api.Close)
	ts := httptest.NewServer(api.Router())
	t.Cleanup(ts.Close)
	return ts, db
}

func jsonClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return &http.Client{Jar: jar}
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, url, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func postJSON(t *testing.T, client *http.Client, url string, body any) *http.Response {
	t.Helper()
	return doJSON(t, client, http.MethodPost, url, body)
}

func patchJSON(t *testing.T, client *http.Client, url string, body any) *http.Response {
	t.Helper()
	return doJSON(t, client, http.MethodPatch, url, body)
}

func deleteReq(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()
	return doJSON(t, client, http.MethodDelete, url, nil)
}

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

// devLoginAs logs in as username (creating it if needed) and returns a client carrying
// the resulting session cookie.
func devLoginAs(t *testing.T, ts *httptest.Server, username string) *http.Client {
	t.Helper()
	client := jsonClient(t)
	resp := postJSON(t, client, ts.URL+"/api/auth/dev-login", map[string]string{"username": username})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dev-login as %s: want 200, got %d", username, resp.StatusCode)
	}
	return client
}

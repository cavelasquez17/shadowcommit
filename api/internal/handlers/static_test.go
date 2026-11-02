package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
)

const (
	indexHTML = "<!doctype html><title>ShadowCommit SPA</title>"
	assetJS   = "console.log('hashed asset')"
	secret    = "TOP-SECRET-OUTSIDE-STATIC-DIR"
)

// staticServer builds a site directory plus a secret file OUTSIDE it, and boots the API
// pointed at the site.
func staticServer(t *testing.T) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	site := filepath.Join(root, "site")
	for path, body := range map[string]string{
		filepath.Join(site, "index.html"):              indexHTML,
		filepath.Join(site, "assets", "app-abc123.js"): assetJS,
		filepath.Join(site, "robots.txt"):              "User-agent: *",
		filepath.Join(root, "secret.txt"):              secret,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ts, _ := newTestServer(t, func(c *config.Config) { c.StaticDir = site })
	return ts
}

func get(t *testing.T, url string, headers map[string]string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestStaticServesIndexAtRootWithoutCaching(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)
	resp, body := get(t, ts.URL+"/", nil)
	if resp.StatusCode != http.StatusOK || body != indexHTML {
		t.Fatalf("GET /: got %d %q", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("index Content-Type = %q, want text/html", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("index Cache-Control = %q, want no-cache (so deploys show up immediately)", cc)
	}
}

func TestStaticHashedAssetsAreCachedForever(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)
	resp, body := get(t, ts.URL+"/assets/app-abc123.js", nil)
	if resp.StatusCode != http.StatusOK || body != assetJS {
		t.Fatalf("asset: got %d %q", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("asset Content-Type = %q, want javascript", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Errorf("asset Cache-Control = %q, want long-lived immutable", cc)
	}
}

func TestStaticOtherRootFilesAreServedButNotCachedForever(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)
	resp, body := get(t, ts.URL+"/robots.txt", nil)
	if resp.StatusCode != http.StatusOK || body != "User-agent: *" {
		t.Fatalf("robots.txt: got %d %q", resp.StatusCode, body)
	}
	if strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Error("only hashed /assets/ files may be immutable")
	}
}

func TestStaticClientRoutesFallBackToIndex(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)
	for _, path := range []string{"/feed", "/t/thr_abc123", "/u/carlos-v", "/jobs"} {
		resp, body := get(t, ts.URL+path, nil)
		if resp.StatusCode != http.StatusOK || body != indexHTML {
			t.Errorf("GET %s: got %d %q, want the SPA shell", path, resp.StatusCode, body)
		}
	}
}

func TestStaticMissingFileWithExtensionIs404NotTheShell(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)
	resp, body := get(t, ts.URL+"/assets/missing-9f.js", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing asset: got %d, want 404", resp.StatusCode)
	}
	if strings.Contains(body, "ShadowCommit SPA") {
		t.Error("a missing .js must not be answered with HTML")
	}
}

func TestStaticNeverShadowsTheAPI(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)

	for _, path := range []string{"/api/nope", "/api/threads/x/y/z", "/api", "/api/"} {
		resp, body := get(t, ts.URL+path, nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: got %d, want 404", path, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "json") || !strings.Contains(body, "not_found") {
			t.Errorf("GET %s should return the JSON 404 envelope, got %q %q", path, ct, body)
		}
	}

	resp, body := get(t, ts.URL+"/healthz", nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "alive") {
		t.Errorf("/healthz must still work, got %d %q", resp.StatusCode, body)
	}
	resp, _ = get(t, ts.URL+"/api/meta", nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("real API routes must still work, got %d", resp.StatusCode)
	}
}

func TestStaticOnlyServesGetAndHead(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)

	resp := postJSON(t, http.DefaultClient, ts.URL+"/", map[string]string{})
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST /: got %d, want 404", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodHead, ts.URL+"/", nil)
	head, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	head.Body.Close()
	if head.StatusCode != http.StatusOK {
		t.Errorf("HEAD /: got %d, want 200", head.StatusCode)
	}
}

func TestStaticCannotEscapeTheDirectory(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)

	// Build raw requests so the Go client does not tidy the path before sending it.
	for _, raw := range []string{
		"/../secret.txt",
		"/%2e%2e/secret.txt",
		"/assets/../../secret.txt",
		"/assets/%2e%2e/%2e%2e/secret.txt",
		"/..%2fsecret.txt",
		"//../secret.txt",
	} {
		req, err := http.NewRequest(http.MethodGet, ts.URL+raw, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(b), secret) {
			t.Errorf("path traversal via %q leaked a file outside the static dir", raw)
		}
	}
}

func TestStaticDirectoryRequestsGetTheShellNeverAListing(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)
	for _, path := range []string{"/assets/", "/assets"} {
		resp, body := get(t, ts.URL+path, nil)
		if resp.StatusCode != http.StatusOK || body != indexHTML {
			t.Errorf("GET %s: got %d %q, want the SPA shell and no directory listing", path, resp.StatusCode, body)
		}
	}
}

func TestStaticNormalisesDotSegmentsInsideTheDirectory(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)
	resp, body := get(t, ts.URL+"/assets/../robots.txt", nil)
	if resp.StatusCode != http.StatusOK || body != "User-agent: *" {
		t.Errorf("/assets/../robots.txt: got %d %q, want robots.txt", resp.StatusCode, body)
	}
}

func TestStaticSupportsConditionalRequests(t *testing.T) {
	t.Parallel()
	ts := staticServer(t)
	first, _ := get(t, ts.URL+"/assets/app-abc123.js", nil)
	lm := first.Header.Get("Last-Modified")
	if lm == "" {
		t.Fatal("static files should carry Last-Modified")
	}
	resp, _ := get(t, ts.URL+"/assets/app-abc123.js", map[string]string{"If-Modified-Since": lm})
	if resp.StatusCode != http.StatusNotModified {
		t.Errorf("conditional GET: got %d, want 304", resp.StatusCode)
	}
}

func TestWithoutStaticDirTheAPIStaysAPIOnly(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	resp, body := get(t, ts.URL+"/", nil)
	if resp.StatusCode != http.StatusNotFound || !strings.Contains(body, "not_found") {
		t.Errorf("GET / without STATIC_DIR: got %d %q, want the JSON 404", resp.StatusCode, body)
	}
}

func TestStaticDirWithoutIndexFailsSafely(t *testing.T) {
	t.Parallel()
	empty := t.TempDir()
	ts, _ := newTestServer(t, func(c *config.Config) { c.StaticDir = empty })
	resp, _ := get(t, ts.URL+"/feed", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("no index.html present: got %d, want 404 rather than a crash", resp.StatusCode)
	}
}

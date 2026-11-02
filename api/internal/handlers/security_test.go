package handlers_test

import (
	"net/http"
	"testing"
)

// TestUpdateProfileRejectsJavascriptURL guards the XSS fix: website_url/linkedin_url
// must be http(s), never javascript:.
func TestUpdateProfileRejectsJavascriptURL(t *testing.T) {
	t.Parallel()
	ts, _, _ := newTestServer(t, nil)
	client := devLoginAs(t, ts, "linkspammer")

	resp := patchJSON(t, client, ts.URL+"/api/profile", map[string]string{
		"website_url": "javascript:alert(document.cookie)",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 rejecting a javascript: website_url, got %d", resp.StatusCode)
	}
}

// TestUpdateProfileAcceptsHTTPSURL guards the fix didn't break legitimate links.
func TestUpdateProfileAcceptsHTTPSURL(t *testing.T) {
	t.Parallel()
	ts, _, _ := newTestServer(t, nil)
	client := devLoginAs(t, ts, "linkeduser")

	resp := patchJSON(t, client, ts.URL+"/api/profile", map[string]string{
		"website_url": "https://example.com/portfolio",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 accepting a real https website_url, got %d", resp.StatusCode)
	}
}

// TestAddRepoRejectsNonHTTPURL guards repo links the same way.
func TestAddRepoRejectsNonHTTPURL(t *testing.T) {
	t.Parallel()
	ts, _, _ := newTestServer(t, nil)
	client := devLoginAs(t, ts, "repospammer")

	resp := postJSON(t, client, ts.URL+"/api/profile/repos", map[string]string{
		"name": "totally-legit", "url": "javascript:alert(1)",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 rejecting a javascript: repo url, got %d", resp.StatusCode)
	}
}

// TestCreateJobRejectsNonHTTPURL guards job apply_url the same way — that page is
// public and reachable without an account.
func TestCreateJobRejectsNonHTTPURL(t *testing.T) {
	t.Parallel()
	ts, _, _ := newTestServer(t, nil)
	client := devLoginAs(t, ts, "jobspammer")

	resp := postJSON(t, client, ts.URL+"/api/jobs", map[string]any{
		"company":   "Acme",
		"title":     "Dev",
		"body":      "come work for us, we swear it is real and not a trap at all",
		"apply_url": "javascript:alert(1)",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 rejecting a javascript: apply_url, got %d", resp.StatusCode)
	}
}

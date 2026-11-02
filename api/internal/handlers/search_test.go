package handlers_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

type searchResult struct {
	Items []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"items"`
	Total int `json:"total"`
}

func newThread(t *testing.T, tsURL string, c *http.Client, title, body string) string {
	t.Helper()
	resp := postJSON(t, c, tsURL+"/api/threads", map[string]any{
		"category": "the-incident", "title": title, "body": body,
	})
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("create thread: got %d", resp.StatusCode)
	}
	out := decodeJSON[struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}](t, resp)
	return out.Thread.ID
}

func search(t *testing.T, tsURL, q string) searchResult {
	t.Helper()
	return decodeJSON[searchResult](t, mustGet(t, tsURL+"/api/threads?q="+url.QueryEscape(q)))
}

func TestThreadSearch(t *testing.T) {
	ts, db := newTestServer(t, nil)
	if err := db.Create(&models.Category{Slug: "the-incident", Name: "The Incident"}).Error; err != nil {
		t.Fatal(err)
	}
	c := devLoginAs(t, ts, "searcher")

	pct := newThread(t, ts.URL, c, "Salary cut of 100% announced", "Body text that is long enough to be valid.")
	plain := newThread(t, ts.URL, c, "Completely unrelated thread", "Another body that is long enough to be valid.")
	withReply := newThread(t, ts.URL, c, "Thread whose reply is searchable", "Yet another body long enough to be valid.")
	resp := postJSON(t, c, ts.URL+"/api/threads/"+withReply+"/posts", map[string]string{"body": "The zanzibar migration ate our weekend"})
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("create post: got %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = postJSON(t, c, ts.URL+"/api/threads/"+withReply+"/posts", map[string]string{"body": "More zanzibar fallout in a second reply"})
	resp.Body.Close()

	t.Run("matches title", func(t *testing.T) {
		if got := search(t, ts.URL, "unrelated"); got.Total != 1 || got.Items[0].ID != plain {
			t.Errorf("want only %s, got %+v", plain, got)
		}
	})
	t.Run("matches reply body", func(t *testing.T) {
		if got := search(t, ts.URL, "zanzibar"); got.Total != 1 || got.Items[0].ID != withReply {
			t.Errorf("want only %s, got %+v", withReply, got)
		}
	})
	t.Run("percent is literal, not a wildcard", func(t *testing.T) {
		if got := search(t, ts.URL, "100%"); got.Total != 1 || got.Items[0].ID != pct {
			t.Errorf("want only %s, got %+v", pct, got)
		}
		if got := search(t, ts.URL, "%"); got.Total != 1 {
			t.Errorf("searching a bare %% should match only the thread containing it, got %d", got.Total)
		}
	})
	t.Run("underscore is literal", func(t *testing.T) {
		if got := search(t, ts.URL, "_"); got.Total != 0 {
			t.Errorf("no thread contains an underscore, got %d", got.Total)
		}
	})
	t.Run("thread with several matching replies is listed once", func(t *testing.T) {
		if got := search(t, ts.URL, "zanzibar"); got.Total != 1 || len(got.Items) != 1 {
			t.Errorf("want exactly 1 thread, got total=%d items=%d", got.Total, len(got.Items))
		}
	})
}

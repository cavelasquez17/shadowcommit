package handlers_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestCreateThreadErrorNamesTheMissingField(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	client := devLoginAs(t, ts, "validation_tester")

	cases := []struct {
		name    string
		body    map[string]any
		mention string
	}{
		{"missing category", map[string]any{"title": "A perfectly fine title", "body": "A body that is long enough to pass."}, "category"},
		{"short title", map[string]any{"category": "the-incident", "title": "short", "body": "A body that is long enough to pass."}, "title"},
		{"short body", map[string]any{"category": "the-incident", "title": "A perfectly fine title", "body": "tiny"}, "body"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := postJSON(t, client, ts.URL+"/api/threads", tc.body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("want 400, got %d", resp.StatusCode)
			}
			msg := decodeJSON[map[string]string](t, resp)["message"]
			if !strings.Contains(strings.ToLower(msg), tc.mention) {
				t.Errorf("message %q should mention %q", msg, tc.mention)
			}
		})
	}
}

package handlers_test

import (
	"net/http"
	"strings"
	"testing"
)

// Editing a thread should only need the fields being changed. The category is not part of an
// update (the handler never reads it), so requiring it made title-only edits fail with a generic
// message.
func TestUpdateThreadDoesNotRequireCategory(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	c := devLoginAs(t, ts, "editor_no_cat")
	id := newThread(t, ts.URL, c, "Original title for the thread", "Body long enough to be a valid thread.")

	resp := patchJSON(t, c, ts.URL+"/api/threads/"+id, map[string]any{
		"title": "Edited title without a category",
		"body":  "Edited body that is also long enough to pass.",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("title-and-body-only edit: got %d, want 200", resp.StatusCode)
	}
	if got := threadByID(t, db, id).Title; got != "Edited title without a category" {
		t.Errorf("title = %q, want the edited one", got)
	}
}

// An invalid update must say which field is wrong, not return a generic message.
func TestUpdateThreadErrorNamesTheField(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	c := devLoginAs(t, ts, "editor_msg")
	id := newThread(t, ts.URL, c, "Original title for the thread", "Body long enough to be a valid thread.")

	resp := patchJSON(t, c, ts.URL+"/api/threads/"+id, map[string]any{"title": "x"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("too-short title: got %d, want 400", resp.StatusCode)
	}
	msg := decodeJSON[map[string]string](t, resp)["message"]
	if !strings.Contains(strings.ToLower(msg), "title") {
		t.Errorf("message %q should name the title", msg)
	}
}

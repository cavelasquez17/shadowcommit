package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

type notifList struct {
	Items []struct {
		Kind   string `json:"kind"`
		IsRead bool   `json:"is_read"`
	} `json:"items"`
	Unread int `json:"unread"`
}

func TestNotificationsAreScopedToUserAndCanBeMarkedRead(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	author := devLoginAs(t, ts, "notif_owner")
	fan := devLoginAs(t, ts, "notif_actor")
	bystander := devLoginAs(t, ts, "notif_bystander")
	id := newThread(t, ts.URL, author, "A thread that collects notifications", "Body long enough to be a valid thread.")
	postJSON(t, fan, ts.URL+"/api/threads/"+id+"/react", map[string]string{"kind": "f"}).Body.Close()
	newPost(t, ts.URL, fan, id, "A reply that notifies the author", "")

	mine := decodeJSON[notifList](t, doJSON(t, author, http.MethodGet, ts.URL+"/api/notifications", nil))
	if len(mine.Items) != 2 || mine.Unread != 2 {
		t.Fatalf("author should have 2 unread notifications, got items=%d unread=%d", len(mine.Items), mine.Unread)
	}
	other := decodeJSON[notifList](t, doJSON(t, bystander, http.MethodGet, ts.URL+"/api/notifications", nil))
	if len(other.Items) != 0 {
		t.Errorf("notifications must never leak across users, bystander saw %d", len(other.Items))
	}

	resp := postJSON(t, bystander, ts.URL+"/api/notifications/read", nil)
	resp.Body.Close()
	again := decodeJSON[notifList](t, doJSON(t, author, http.MethodGet, ts.URL+"/api/notifications", nil))
	if again.Unread != 2 {
		t.Errorf("someone else marking read must not affect the author, unread=%d", again.Unread)
	}

	resp = postJSON(t, author, ts.URL+"/api/notifications/read", nil)
	resp.Body.Close()
	done := decodeJSON[notifList](t, doJSON(t, author, http.MethodGet, ts.URL+"/api/notifications", nil))
	if done.Unread != 0 || len(done.Items) != 2 {
		t.Errorf("after mark-read want unread=0 and history kept, got unread=%d items=%d", done.Unread, len(done.Items))
	}

	resp = doJSON(t, &http.Client{}, http.MethodGet, ts.URL+"/api/notifications", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous notifications: got %d, want 401", resp.StatusCode)
	}
}

type inviteResp struct {
	Invite struct {
		Code string `json:"code"`
	} `json:"invite"`
	URL string `json:"url"`
}

func TestInviteLifecycle(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	owner := devLoginAs(t, ts, "inviter")

	created := decodeJSON[inviteResp](t, postJSON(t, owner, ts.URL+"/api/invites", map[string]string{"note": "for my ex-teammate"}))
	code := created.Invite.Code
	if len(code) != 8 {
		t.Fatalf("invite code should have 8 characters, got %q", code)
	}
	if want := "http://localhost:5173/join?code=" + code; created.URL != want {
		t.Errorf("invite url = %q, want %q", created.URL, want)
	}

	check := func(c string) map[string]any {
		return decodeJSON[map[string]any](t, mustGet(t, ts.URL+"/api/invites/check?code="+c))
	}
	if v := check(code); v["valid"] != true || v["note"] != "for my ex-teammate" {
		t.Errorf("fresh invite should be valid with its note, got %v", v)
	}
	if v := check("lowercase"); v["valid"] != false {
		t.Errorf("unknown code must be invalid, got %v", v)
	}

	db.Model(&models.Invite{}).Where("code = ?", code).Update("expires_at", time.Now().Add(-time.Hour))
	if v := check(code); v["valid"] != false {
		t.Errorf("expired invite must be invalid, got %v", v)
	}

	list := decodeJSON[struct {
		Items []struct{ Code string } `json:"items"`
	}](t, doJSON(t, owner, http.MethodGet, ts.URL+"/api/invites", nil))
	if len(list.Items) != 1 {
		t.Errorf("owner should list 1 invite, got %d", len(list.Items))
	}
	stranger := devLoginAs(t, ts, "invite_stranger")
	theirs := decodeJSON[struct {
		Items []struct{ Code string } `json:"items"`
	}](t, doJSON(t, stranger, http.MethodGet, ts.URL+"/api/invites", nil))
	if len(theirs.Items) != 0 {
		t.Errorf("invites must be private to their creator, stranger saw %d", len(theirs.Items))
	}
}

func TestInviteCodesAreCaseInsensitiveOnCheckAndCapAtTen(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	owner := devLoginAs(t, ts, "inviter_cap")

	var first string
	for i := 0; i < 10; i++ {
		code := decodeJSON[inviteResp](t, postJSON(t, owner, ts.URL+"/api/invites", nil)).Invite.Code
		if i == 0 {
			first = code
		}
	}
	resp := postJSON(t, owner, ts.URL+"/api/invites", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("11th active invite: got %d, want 429", resp.StatusCode)
	}

	lower := []byte(first)
	for i := range lower {
		if lower[i] >= 'A' && lower[i] <= 'Z' {
			lower[i] += 'a' - 'A'
		}
	}
	v := decodeJSON[map[string]any](t, mustGet(t, ts.URL+"/api/invites/check?code="+string(lower)))
	if v["valid"] != true {
		t.Errorf("codes should be accepted regardless of case, got %v", v)
	}
}

func TestMentionsListShowsWhoTaggedYou(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	author := devLoginAs(t, ts, "tagger-one")
	target := devLoginAs(t, ts, "tagged-one")
	id := newThread(t, ts.URL, author, "A thread that tags somebody", "Body long enough to be a valid thread.")
	newPost(t, ts.URL, author, id, "ping @tagged-one please", "")

	out := decodeJSON[struct {
		Items []struct {
			ByUsername  string `json:"by_username"`
			ThreadID    string `json:"thread_id"`
			ThreadTitle string `json:"thread_title"`
		} `json:"items"`
	}](t, doJSON(t, target, http.MethodGet, ts.URL+"/api/mentions", nil))
	if len(out.Items) != 1 || out.Items[0].ByUsername != "tagger-one" || out.Items[0].ThreadID != id {
		t.Errorf("want one mention by tagger-one in %s, got %+v", id, out.Items)
	}
	none := decodeJSON[struct {
		Items []struct{} `json:"items"`
	}](t, doJSON(t, author, http.MethodGet, ts.URL+"/api/mentions", nil))
	if len(none.Items) != 0 {
		t.Errorf("the tagger was not mentioned, got %d", len(none.Items))
	}
}

func TestSearchUsersAutocomplete(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	for _, n := range []string{"maria", "mario", "juan"} {
		devLoginAs(t, ts, n)
	}
	names := func(q string) []string {
		out := decodeJSON[struct {
			Items []struct {
				Username string `json:"username"`
			} `json:"items"`
		}](t, mustGet(t, ts.URL+"/api/users/search?q="+q))
		var r []string
		for _, i := range out.Items {
			r = append(r, i.Username)
		}
		return r
	}
	if got := names("mar"); len(got) != 2 {
		t.Errorf("q=mar should match maria and mario, got %v", got)
	}
	if got := names("MARIA"); len(got) != 1 || got[0] != "maria" {
		t.Errorf("search is case-insensitive, got %v", got)
	}
	if got := names(""); len(got) != 0 {
		t.Errorf("empty query must return nothing, got %v", got)
	}
}

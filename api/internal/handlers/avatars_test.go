package handlers_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAvatarEndpointServesAStableSVG(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)

	resp := mustGet(t, ts.URL+"/api/avatars/alice")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "max-age") {
		t.Errorf("Cache-Control = %q, want a max-age so browsers keep it", cc)
	}
	if !strings.HasPrefix(string(body), "<svg") {
		t.Errorf("body does not start with <svg: %.40q", body)
	}

	again := mustGet(t, ts.URL+"/api/avatars/alice")
	body2, _ := io.ReadAll(again.Body)
	again.Body.Close()
	if string(body) != string(body2) {
		t.Error("the same username must get the same avatar")
	}
}

func TestAvatarEndpointRejectsOddNames(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	for _, name := range []string{"has%20space", "x%3Cscript%3E", strings.Repeat("a", 60)} {
		resp := mustGet(t, ts.URL+"/api/avatars/"+name)
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("name %q: got %d, want 400", name, resp.StatusCode)
		}
	}
}

func TestNewUsersGetLocalAvatars(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	devLoginAs(t, ts, "fresh_avatar")
	u := userByName(t, db, "fresh_avatar")
	if strings.Contains(u.AvatarURL, "dicebear") || strings.HasPrefix(u.AvatarURL, "http") {
		t.Errorf("new user avatar_url = %q, must be a local API path", u.AvatarURL)
	}
	if u.AvatarURL != "/api/avatars/fresh_avatar" {
		t.Errorf("avatar_url = %q, want /api/avatars/fresh_avatar", u.AvatarURL)
	}
}

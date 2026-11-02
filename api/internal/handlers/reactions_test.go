package handlers_test

import (
	"net/http"
	"testing"
)

type reactionsResp struct {
	Reactions []struct {
		Kind  string `json:"kind"`
		Count int    `json:"count"`
		Mine  bool   `json:"mine"`
	} `json:"reactions"`
}

func (r reactionsResp) count(kind string) (int, bool) {
	for _, x := range r.Reactions {
		if x.Kind == kind {
			return x.Count, x.Mine
		}
	}
	return -1, false
}

func TestReactionToggleOnThreadUpdatesCountersAndReputation(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	author := devLoginAs(t, ts, "react_author")
	fan := devLoginAs(t, ts, "react_fan")
	id := newThread(t, ts.URL, author, "A thread worth reacting to", "Body long enough to be a valid thread.")

	rep0 := userByName(t, db, "react_author").Reputation

	on := decodeJSON[reactionsResp](t, postJSON(t, fan, ts.URL+"/api/threads/"+id+"/react", map[string]string{"kind": "f"}))
	if c, mine := on.count("f"); c != 1 || !mine {
		t.Fatalf("after reacting want count=1 mine=true, got count=%d mine=%v", c, mine)
	}
	if got := threadByID(t, db, id).ReactionCount; got != 1 {
		t.Errorf("thread reaction_count = %d, want 1", got)
	}
	if got := userByName(t, db, "react_author").Reputation; got != rep0+1 {
		t.Errorf("author reputation = %d, want %d", got, rep0+1)
	}

	off := decodeJSON[reactionsResp](t, postJSON(t, fan, ts.URL+"/api/threads/"+id+"/react", map[string]string{"kind": "f"}))
	if c, mine := off.count("f"); c != 0 || mine {
		t.Fatalf("reacting twice must remove it, got count=%d mine=%v", c, mine)
	}
	if got := threadByID(t, db, id).ReactionCount; got != 0 {
		t.Errorf("thread reaction_count after toggle off = %d, want 0", got)
	}
	if got := userByName(t, db, "react_author").Reputation; got != rep0 {
		t.Errorf("author reputation after toggle off = %d, want %d", got, rep0)
	}
}

func TestReactionKindsAreIndependentAndMineIsPerViewer(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	a := devLoginAs(t, ts, "kinds_a")
	b := devLoginAs(t, ts, "kinds_b")
	id := newThread(t, ts.URL, a, "Kinds of reactions here", "Body long enough to be a valid thread.")

	postJSON(t, a, ts.URL+"/api/threads/"+id+"/react", map[string]string{"kind": "f"}).Body.Close()
	resp := decodeJSON[reactionsResp](t, postJSON(t, b, ts.URL+"/api/threads/"+id+"/react", map[string]string{"kind": "laugh"}))

	if c, mine := resp.count("f"); c != 1 || mine {
		t.Errorf("viewer b: f should be count=1 mine=false, got %d/%v", c, mine)
	}
	if c, mine := resp.count("laugh"); c != 1 || !mine {
		t.Errorf("viewer b: laugh should be count=1 mine=true, got %d/%v", c, mine)
	}
	if got := threadByID(t, db, id).ReactionCount; got != 2 {
		t.Errorf("thread reaction_count = %d, want 2", got)
	}
}

func TestReactionOnPost(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	a := devLoginAs(t, ts, "post_react_a")
	b := devLoginAs(t, ts, "post_react_b")
	tid := newThread(t, ts.URL, a, "A thread with a reply", "Body long enough to be a valid thread.")
	pid := newPost(t, ts.URL, a, tid, "A reply to react to", "")

	resp := decodeJSON[reactionsResp](t, postJSON(t, b, ts.URL+"/api/posts/"+pid+"/react", map[string]string{"kind": "f"}))
	if c, mine := resp.count("f"); c != 1 || !mine {
		t.Fatalf("want count=1 mine=true on post, got %d/%v", c, mine)
	}
}

func TestReactionRejectsBadInput(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	c := devLoginAs(t, ts, "react_bad")
	id := newThread(t, ts.URL, c, "Thread for bad reactions", "Body long enough to be a valid thread.")

	cases := []struct {
		name, url string
		body      any
		want      int
	}{
		{"unknown kind", "/api/threads/" + id + "/react", map[string]string{"kind": "nope"}, http.StatusBadRequest},
		{"missing kind", "/api/threads/" + id + "/react", map[string]string{}, http.StatusBadRequest},
		{"unknown thread", "/api/threads/thr_doesnotexist/react", map[string]string{"kind": "f"}, http.StatusNotFound},
		{"unknown post", "/api/posts/pst_doesnotexist/react", map[string]string{"kind": "f"}, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := postJSON(t, c, ts.URL+tc.url, tc.body)
			defer resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Errorf("got %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}

	anon := &http.Client{}
	resp := postJSON(t, anon, ts.URL+"/api/threads/"+id+"/react", map[string]string{"kind": "f"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous reaction: got %d, want 401", resp.StatusCode)
	}
}

func TestReactionNotifiesAuthorButNotSelf(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	author := devLoginAs(t, ts, "notif_author")
	fan := devLoginAs(t, ts, "notif_fan")
	id := newThread(t, ts.URL, author, "A thread that gets a reaction", "Body long enough to be a valid thread.")
	authorID := userByName(t, db, "notif_author").ID

	postJSON(t, author, ts.URL+"/api/threads/"+id+"/react", map[string]string{"kind": "f"}).Body.Close()
	if n := notificationCount(t, db, authorID, "reaction"); n != 0 {
		t.Errorf("reacting to your own thread must not notify you, got %d", n)
	}
	postJSON(t, fan, ts.URL+"/api/threads/"+id+"/react", map[string]string{"kind": "f"}).Body.Close()
	if n := notificationCount(t, db, authorID, "reaction"); n != 1 {
		t.Errorf("author should have 1 reaction notification, got %d", n)
	}
}

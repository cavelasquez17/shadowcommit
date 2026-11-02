package handlers_test

import (
	"net/http"
	"testing"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

func TestCreatePostUpdatesCountersAndNotifiesThreadAuthor(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	author := devLoginAs(t, ts, "post_author")
	replier := devLoginAs(t, ts, "post_replier")
	tid := newThread(t, ts.URL, author, "A thread that will get replies", "Body long enough to be a valid thread.")
	authorID := userByName(t, db, "post_author").ID

	newPost(t, ts.URL, replier, tid, "First reply from someone else", "")

	th := threadByID(t, db, tid)
	if th.PostCount != 1 || th.LastPostAt == nil {
		t.Errorf("thread post_count=%d last_post_at=%v, want 1 and set", th.PostCount, th.LastPostAt)
	}
	if got := userByName(t, db, "post_replier").PostCount; got != 1 {
		t.Errorf("replier post_count = %d, want 1", got)
	}
	if n := notificationCount(t, db, authorID, "reply"); n != 1 {
		t.Errorf("thread author should get 1 reply notification, got %d", n)
	}

	newPost(t, ts.URL, author, tid, "The author answering their own thread", "")
	if n := notificationCount(t, db, authorID, "reply"); n != 1 {
		t.Errorf("replying to your own thread must not notify you, got %d notifications", n)
	}
}

func TestCreatePostValidationAndErrors(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	c := devLoginAs(t, ts, "post_validation")
	tid := newThread(t, ts.URL, c, "Thread for validation tests", "Body long enough to be a valid thread.")

	cases := []struct {
		name, url string
		body      map[string]any
		want      int
	}{
		{"body too short", "/api/threads/" + tid + "/posts", map[string]any{"body": "x"}, http.StatusBadRequest},
		{"missing body", "/api/threads/" + tid + "/posts", map[string]any{}, http.StatusBadRequest},
		{"unknown thread", "/api/threads/thr_nope/posts", map[string]any{"body": "hello there"}, http.StatusNotFound},
		{"unknown parent", "/api/threads/" + tid + "/posts", map[string]any{"body": "hello there", "parent_id": "pst_nope"}, http.StatusBadRequest},
	}
	t.Run("rejected", func(t *testing.T) {
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
	})
	if got := threadByID(t, db, tid).PostCount; got != 0 {
		t.Errorf("rejected replies must not bump post_count, got %d", got)
	}

	resp := postJSON(t, &http.Client{}, ts.URL+"/api/threads/"+tid+"/posts", map[string]any{"body": "anonymous hello"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous reply: got %d, want 401", resp.StatusCode)
	}
}

func TestCreatePostRejectedOnLockedThread(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	c := devLoginAs(t, ts, "locked_user")
	tid := newThread(t, ts.URL, c, "A thread about to be locked", "Body long enough to be a valid thread.")
	if err := db.Model(&models.Thread{}).Where("public_id = ?", tid).Update("is_locked", true).Error; err != nil {
		t.Fatal(err)
	}

	resp := postJSON(t, c, ts.URL+"/api/threads/"+tid+"/posts", map[string]any{"body": "can I still talk?"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("reply to locked thread: got %d, want 409", resp.StatusCode)
	}
	if got := threadByID(t, db, tid).PostCount; got != 0 {
		t.Errorf("locked thread post_count = %d, want 0", got)
	}
}

func TestNestedRepliesFlattenToTopLevelParent(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	c := devLoginAs(t, ts, "nester")
	tid := newThread(t, ts.URL, c, "A thread with nested replies", "Body long enough to be a valid thread.")

	top := newPost(t, ts.URL, c, tid, "Top level reply", "")
	child := newPost(t, ts.URL, c, tid, "Reply to the top level", top)
	grandchild := newPost(t, ts.URL, c, tid, "Reply to the reply", child)

	var g models.Post
	if err := db.Where("public_id = ?", grandchild).First(&g).Error; err != nil {
		t.Fatal(err)
	}
	var topPost models.Post
	db.Where("public_id = ?", top).First(&topPost)
	if g.ParentID == nil || *g.ParentID != topPost.ID {
		t.Errorf("grandchild must attach to the top-level post (%d), got parent %v", topPost.ID, g.ParentID)
	}
}

func TestReplyParentMustBelongToSameThread(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	c := devLoginAs(t, ts, "cross_thread")
	t1 := newThread(t, ts.URL, c, "First thread of two here", "Body long enough to be a valid thread.")
	t2 := newThread(t, ts.URL, c, "Second thread of two here", "Body long enough to be a valid thread.")
	p1 := newPost(t, ts.URL, c, t1, "A reply in the first thread", "")

	resp := postJSON(t, c, ts.URL+"/api/threads/"+t2+"/posts", map[string]any{"body": "wrong thread parent", "parent_id": p1})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("parent from another thread: got %d, want 400", resp.StatusCode)
	}
}

func TestUpdatePostPermissions(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	owner := devLoginAs(t, ts, "edit_owner")
	other := devLoginAs(t, ts, "edit_other")
	admin := devLoginAs(t, ts, "edit_admin")
	makeAdmin(t, db, "edit_admin")
	tid := newThread(t, ts.URL, owner, "A thread with an editable reply", "Body long enough to be a valid thread.")
	pid := newPost(t, ts.URL, owner, tid, "Original reply text", "")

	cases := []struct {
		name   string
		client *http.Client
		want   int
	}{
		{"owner", owner, http.StatusOK},
		{"admin", admin, http.StatusOK},
		{"other member", other, http.StatusForbidden},
		{"anonymous", &http.Client{}, http.StatusUnauthorized},
	}
	for _, tc := range cases { //nolint:paralleltest // cases edit one post in order and the final state is asserted
		t.Run(tc.name, func(t *testing.T) {
			resp := patchJSON(t, tc.client, ts.URL+"/api/posts/"+pid, map[string]any{"body": "Edited by " + tc.name})
			defer resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Errorf("got %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}

	var p models.Post
	db.Where("public_id = ?", pid).First(&p)
	if p.Body != "Edited by admin" {
		t.Errorf("body = %q: forbidden edits must not change it, last allowed edit was the admin's", p.Body)
	}

	resp := patchJSON(t, owner, ts.URL+"/api/posts/pst_missing", map[string]any{"body": "ghost"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("editing a missing post: got %d, want 404", resp.StatusCode)
	}
}

func TestDeletePostPermissionsAndCounters(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	owner := devLoginAs(t, ts, "del_owner")
	other := devLoginAs(t, ts, "del_other")
	admin := devLoginAs(t, ts, "del_admin")
	makeAdmin(t, db, "del_admin")
	tid := newThread(t, ts.URL, owner, "A thread with deletable replies", "Body long enough to be a valid thread.")
	p1 := newPost(t, ts.URL, owner, tid, "Reply that the owner deletes", "")
	p2 := newPost(t, ts.URL, owner, tid, "Reply that an admin deletes", "")

	resp := deleteReq(t, other, ts.URL+"/api/posts/"+p1)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("other member deleting: got %d, want 403", resp.StatusCode)
	}
	if got := threadByID(t, db, tid).PostCount; got != 2 {
		t.Fatalf("a forbidden delete must not change post_count, got %d", got)
	}

	resp = deleteReq(t, owner, ts.URL+"/api/posts/"+p1)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner deleting: got %d, want 200", resp.StatusCode)
	}
	resp = deleteReq(t, admin, ts.URL+"/api/posts/"+p2)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin deleting: got %d, want 200", resp.StatusCode)
	}

	if got := threadByID(t, db, tid).PostCount; got != 0 {
		t.Errorf("thread post_count = %d, want 0 after both deletes", got)
	}
	if got := userByName(t, db, "del_owner").PostCount; got != 0 {
		t.Errorf("author post_count = %d, want 0", got)
	}

	resp = deleteReq(t, owner, ts.URL+"/api/posts/"+p1)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("deleting twice: got %d, want 404", resp.StatusCode)
	}
}

func TestMentionsNotifyMentionedUser(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	author := devLoginAs(t, ts, "mention_author")
	devLoginAs(t, ts, "mention-target")
	tid := newThread(t, ts.URL, author, "A thread with a mention in it", "Body long enough to be a valid thread.")
	targetID := userByName(t, db, "mention-target").ID

	newPost(t, ts.URL, author, tid, "hey @mention-target look at this", "")
	if n := notificationCount(t, db, targetID, "mention"); n != 1 {
		t.Errorf("mentioned user should get 1 mention notification, got %d", n)
	}
}

func TestMentionsIgnoreSelfUnknownAndDuplicates(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	ensureCategory(t, db)
	author := devLoginAs(t, ts, "mention-self")
	devLoginAs(t, ts, "mention-once")
	tid := newThread(t, ts.URL, author, "A thread for mention edge cases", "Body long enough to be a valid thread.")
	selfID := userByName(t, db, "mention-self").ID
	onceID := userByName(t, db, "mention-once").ID

	newPost(t, ts.URL, author, tid, "tagging @mention-self, @ghost-user, @mention-once and again @mention-once", "")

	if n := notificationCount(t, db, selfID, "mention"); n != 0 {
		t.Errorf("self-mention must not notify, got %d", n)
	}
	if n := notificationCount(t, db, onceID, "mention"); n != 1 {
		t.Errorf("a user mentioned twice in one post should be notified once, got %d", n)
	}
}

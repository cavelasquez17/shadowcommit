package handlers_test

import (
	"net/http"
	"testing"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

type threadView struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Sarcasm   int    `json:"sarcasm"`
	ViewCount int    `json:"view_count"`
	CanEdit   bool   `json:"can_edit"`
	Tags      []struct {
		Slug       string `json:"slug"`
		UsageCount int    `json:"usage_count"`
	} `json:"tags"`
}

type threadDetail struct {
	Thread threadView `json:"thread"`
	Posts  []struct {
		ID      string `json:"id"`
		CanEdit bool   `json:"can_edit"`
	} `json:"posts"`
}

func fetchThread(t *testing.T, tsURL string, c *http.Client, id string) (int, threadDetail) {
	t.Helper()
	resp := doJSON(t, c, http.MethodGet, tsURL+"/api/threads/"+id, nil)
	status := resp.StatusCode
	if status != http.StatusOK {
		resp.Body.Close()
		return status, threadDetail{}
	}
	return status, decodeJSON[threadDetail](t, resp)
}

func editBody(title string, extra map[string]any) map[string]any {
	b := map[string]any{
		"category": "the-incident", "title": title,
		"body": "An edited body that is certainly long enough.",
	}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func TestGetThreadReturnsPostsInOrderAndCountsViews(t *testing.T) {
	t.Parallel()
	ts, db, _ := newTestServer(t, nil)
	ensureCategory(t, db)
	author := devLoginAs(t, ts, "get_author")
	id := newThread(t, ts.URL, author, "A thread that gets read", "Body long enough to be a valid thread.")
	p1 := newPost(t, ts.URL, author, id, "First reply in order", "")
	p2 := newPost(t, ts.URL, author, id, "Second reply in order", "")

	_, d := fetchThread(t, ts.URL, &http.Client{}, id) // anonymous readers are fine
	if len(d.Posts) != 2 || d.Posts[0].ID != p1 || d.Posts[1].ID != p2 {
		t.Fatalf("posts must be in chronological order [%s %s], got %+v", p1, p2, d.Posts)
	}
	if d.Thread.CanEdit {
		t.Error("an anonymous viewer must not be able to edit")
	}
	if d.Thread.ViewCount != 1 {
		t.Errorf("first view should report view_count=1, got %d", d.Thread.ViewCount)
	}
	_, again := fetchThread(t, ts.URL, &http.Client{}, id)
	if again.Thread.ViewCount != 2 {
		t.Errorf("second view should report 2, got %d", again.Thread.ViewCount)
	}

	_, mine := fetchThread(t, ts.URL, author, id)
	if !mine.Thread.CanEdit || !mine.Posts[0].CanEdit {
		t.Error("the author must see can_edit=true on their thread and posts")
	}

	if status, _ := fetchThread(t, ts.URL, author, "thr_unknown"); status != http.StatusNotFound {
		t.Errorf("unknown thread: got %d, want 404", status)
	}
}

func TestUpdateThread(t *testing.T) {
	t.Parallel()
	ts, db, _ := newTestServer(t, nil)
	ensureCategory(t, db)
	owner := devLoginAs(t, ts, "upd_owner")
	other := devLoginAs(t, ts, "upd_other")
	admin := devLoginAs(t, ts, "upd_admin")
	makeAdmin(t, db, "upd_admin")
	id := newThread(t, ts.URL, owner, "Original thread title", "Body long enough to be a valid thread.")

	cases := []struct {
		name   string
		client *http.Client
		path   string
		body   map[string]any
		want   int
	}{
		{"other member", other, id, editBody("Hijacked thread title", nil), http.StatusForbidden},
		{"anonymous", &http.Client{}, id, editBody("Anonymous thread edit", nil), http.StatusUnauthorized},
		{"unknown thread", owner, "thr_unknown", editBody("Edit of a ghost thread", nil), http.StatusNotFound},
		{"invalid payload", owner, id, map[string]any{"title": "x"}, http.StatusBadRequest},
		{"admin", admin, id, editBody("Edited by an admin here", nil), http.StatusOK},
		{"owner", owner, id, editBody("Edited by the owner here", map[string]any{"sarcasm": 250}), http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := patchJSON(t, tc.client, ts.URL+"/api/threads/"+tc.path, tc.body)
			defer resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Errorf("got %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}

	th := threadByID(t, db, id)
	if th.Title != "Edited by the owner here" || th.Slug != "edited-by-the-owner-here" {
		t.Errorf("title/slug = %q/%q: forbidden edits must not stick and the slug must follow the title", th.Title, th.Slug)
	}
	if th.Sarcasm != 100 {
		t.Errorf("sarcasm = %d, want 250 clamped to 100", th.Sarcasm)
	}
}

func TestThreadTagsAreDeduplicatedCappedAndCounted(t *testing.T) {
	t.Parallel()
	ts, db, _ := newTestServer(t, nil)
	ensureCategory(t, db)
	c := devLoginAs(t, ts, "tagger")

	resp := postJSON(t, c, ts.URL+"/api/threads", map[string]any{
		"category": "the-incident", "title": "A thread with lots of tags", "body": "Body long enough to be a valid thread.",
		"tags": []string{"Layoffs", "layoffs", " LAYOFFS ", "Go", "Rust", "AI", "Burnout", "Seventh", "!!!"},
	})
	got := decodeJSON[struct {
		Thread threadView `json:"thread"`
	}](t, resp).Thread
	if len(got.Tags) != 5 {
		t.Fatalf("want 5 tags (deduplicated, capped), got %d: %+v", len(got.Tags), got.Tags)
	}
	if got.Tags[0].Slug != "layoffs" {
		t.Errorf("first tag = %q, want layoffs (case-insensitive dedupe)", got.Tags[0].Slug)
	}

	newThread2 := postJSON(t, c, ts.URL+"/api/threads", map[string]any{
		"category": "the-incident", "title": "Another layoffs thread", "body": "Body long enough to be a valid thread.",
		"tags": []string{"layoffs"},
	})
	newThread2.Body.Close()
	var tag models.Tag
	if err := db.Where("slug = ?", "layoffs").First(&tag).Error; err != nil {
		t.Fatal(err)
	}
	if tag.UsageCount != 2 {
		t.Errorf("layoffs usage_count = %d, want 2 after two threads", tag.UsageCount)
	}

	list := mustGet(t, ts.URL+"/api/tags")
	tags := decodeJSON[struct {
		Items []struct {
			Slug string `json:"slug"`
		} `json:"items"`
	}](t, list)
	if len(tags.Items) == 0 || tags.Items[0].Slug != "layoffs" {
		t.Errorf("tags must be ordered by usage, most used first; got %+v", tags.Items)
	}
}

func TestListCategoriesRespectsSortOrder(t *testing.T) {
	t.Parallel()
	ts, db, _ := newTestServer(t, nil)
	db.Create(&models.Category{Slug: "zeta", Name: "Zeta", SortOrder: 1})
	db.Create(&models.Category{Slug: "alpha", Name: "Alpha", SortOrder: 2})
	db.Create(&models.Category{Slug: "beta", Name: "Beta", SortOrder: 2})

	out := decodeJSON[struct {
		Items []struct {
			Slug string `json:"slug"`
		} `json:"items"`
	}](t, mustGet(t, ts.URL+"/api/categories"))
	var order []string
	for _, i := range out.Items {
		order = append(order, i.Slug)
	}
	want := []string{"zeta", "alpha", "beta"} // sort_order, then name
	if len(order) != 3 || order[0] != want[0] || order[1] != want[1] || order[2] != want[2] {
		t.Errorf("categories order = %v, want %v", order, want)
	}
}

func TestListThreadsFiltersAndSorts(t *testing.T) {
	t.Parallel()
	ts, db, _ := newTestServer(t, nil)
	ensureCategory(t, db)
	db.Create(&models.Category{Slug: "layoffs", Name: "Layoffs"})
	alice := devLoginAs(t, ts, "filter_alice")
	bob := devLoginAs(t, ts, "filter_bob")

	mk := func(c *http.Client, title, cat, lang string, tags []string) string {
		resp := postJSON(t, c, ts.URL+"/api/threads", map[string]any{
			"category": cat, "title": title, "body": "Body long enough to be a valid thread.",
			"language": lang, "tags": tags,
		})
		return decodeJSON[struct {
			Thread struct {
				ID string `json:"id"`
			} `json:"thread"`
		}](t, resp).Thread.ID
	}
	a1 := mk(alice, "Alice thread in English", "the-incident", "en", []string{"go"})
	b1 := mk(bob, "Bob thread in Spanish", "layoffs", "es", []string{"rust"})
	b2 := mk(bob, "Bob second thread here", "layoffs", "en", nil)
	postJSON(t, alice, ts.URL+"/api/threads/"+b1+"/react", map[string]string{"kind": "f"}).Body.Close()

	ids := func(q string) []string {
		l := decodeJSON[searchResult](t, mustGet(t, ts.URL+"/api/threads"+q))
		var out []string
		for _, i := range l.Items {
			out = append(out, i.ID)
		}
		return out
	}
	only := func(q, want string) {
		t.Helper()
		if got := ids(q); len(got) != 1 || got[0] != want {
			t.Errorf("%s: want only %s, got %v", q, want, got)
		}
	}
	only("?category=the-incident", a1)
	only("?language=es", b1)
	only("?author=filter_alice", a1)
	only("?tag=rust", b1)
	if got := ids("?category=layoffs"); len(got) != 2 {
		t.Errorf("category=layoffs: want 2 threads, got %v", got)
	}
	if got := ids("?sort=top"); len(got) != 3 || got[0] != b1 {
		t.Errorf("sort=top: the reacted thread must come first, got %v", got)
	}
	if got := ids("?sort=new"); len(got) != 3 || got[0] != b2 {
		t.Errorf("sort=new: the newest thread must come first, got %v", got)
	}
	if got := ids("?category=nope"); len(got) != 0 {
		t.Errorf("unknown category should yield nothing, got %v", got)
	}
}

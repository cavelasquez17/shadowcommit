package handlers_test

import (
	"net/http"
	"testing"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

// TestDeleteThreadDecrementsCounters guards the transaction-wrap fix: deleting a thread
// must consistently decrement both the author's and the category's thread_count in the
// same write as the soft delete.
func TestDeleteThreadDecrementsCounters(t *testing.T) {
	t.Parallel()
	ts, db, _ := newTestServer(t, nil)
	client := devLoginAs(t, ts, "threadauthor")

	cat := models.Category{Slug: "layoffs", Name: "Layoffs"}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("seed category: %v", err)
	}

	created := postJSON(t, client, ts.URL+"/api/threads", map[string]any{
		"title":    "My whole team is gone",
		"body":     "This body is definitely more than twenty characters long, guaranteed to pass.",
		"category": "layoffs",
	})
	body := decodeJSON[map[string]any](t, created)
	thread, ok := body["thread"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected create response: %v", body)
	}
	threadID, _ := thread["id"].(string)
	if threadID == "" {
		t.Fatalf("thread response missing id: %v", body)
	}

	var author models.User
	if err := db.Where("username = ?", "threadauthor").First(&author).Error; err != nil {
		t.Fatalf("reload author: %v", err)
	}
	if author.ThreadCount != 1 {
		t.Fatalf("want thread_count=1 after create, got %d", author.ThreadCount)
	}

	resp := deleteReq(t, client, ts.URL+"/api/threads/"+threadID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 deleting own thread, got %d", resp.StatusCode)
	}

	if err := db.Where("username = ?", "threadauthor").First(&author).Error; err != nil {
		t.Fatalf("reload author: %v", err)
	}
	if author.ThreadCount != 0 {
		t.Fatalf("want thread_count=0 after delete, got %d", author.ThreadCount)
	}
	var gotCat models.Category
	if err := db.Where("slug = ?", "layoffs").First(&gotCat).Error; err != nil {
		t.Fatalf("reload category: %v", err)
	}
	if gotCat.ThreadCount != 0 {
		t.Fatalf("want category thread_count=0 after delete, got %d", gotCat.ThreadCount)
	}
}

// TestDeletePostDecrementsCounters mirrors the thread case for replies: thread.post_count
// and author.post_count must both come back down after a delete.
func TestDeletePostDecrementsCounters(t *testing.T) {
	t.Parallel()
	ts, db, _ := newTestServer(t, nil)
	client := devLoginAs(t, ts, "postauthor")

	cat := models.Category{Slug: "restructuring", Name: "Restructuring"}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("seed category: %v", err)
	}
	threadResp := postJSON(t, client, ts.URL+"/api/threads", map[string]any{
		"title":    "Reorg eliminated my whole function",
		"body":     "This body is definitely more than twenty characters long, guaranteed to pass.",
		"category": "restructuring",
	})
	threadBody := decodeJSON[map[string]any](t, threadResp)
	thread, ok := threadBody["thread"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected create response: %v", threadBody)
	}
	threadID, _ := thread["id"].(string)

	postResp := postJSON(t, client, ts.URL+"/api/threads/"+threadID+"/posts", map[string]any{
		"body": "Same. Press F.",
	})
	postBody := decodeJSON[map[string]any](t, postResp)
	post, ok := postBody["post"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected post response: %v", postBody)
	}
	postID, _ := post["id"].(string)
	if postID == "" {
		t.Fatalf("post response missing id: %v", postBody)
	}

	var author models.User
	if err := db.Where("username = ?", "postauthor").First(&author).Error; err != nil {
		t.Fatalf("reload author: %v", err)
	}
	if author.PostCount != 1 {
		t.Fatalf("want post_count=1 after reply, got %d", author.PostCount)
	}

	resp := deleteReq(t, client, ts.URL+"/api/posts/"+postID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 deleting own post, got %d", resp.StatusCode)
	}

	if err := db.Where("username = ?", "postauthor").First(&author).Error; err != nil {
		t.Fatalf("reload author: %v", err)
	}
	if author.PostCount != 0 {
		t.Fatalf("want post_count=0 after delete, got %d", author.PostCount)
	}
	var gotThread models.Thread
	if err := db.Where("public_id = ?", threadID).First(&gotThread).Error; err != nil {
		t.Fatalf("reload thread: %v", err)
	}
	if gotThread.PostCount != 0 {
		t.Fatalf("want thread post_count=0 after delete, got %d", gotThread.PostCount)
	}
}

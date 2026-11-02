package handlers_test

import (
	"net/http"
	"testing"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

// ensureCategory makes sure the default test category exists.
func ensureCategory(t *testing.T, db *gorm.DB) {
	t.Helper()
	c := models.Category{Slug: "the-incident", Name: "The Incident"}
	if err := db.Where("slug = ?", c.Slug).FirstOrCreate(&c).Error; err != nil {
		t.Fatalf("ensure category: %v", err)
	}
}

// userByName reloads a user so tests can assert on counters and reputation.
func userByName(t *testing.T, db *gorm.DB, name string) models.User {
	t.Helper()
	var u models.User
	if err := db.Where("username = ?", name).First(&u).Error; err != nil {
		t.Fatalf("load user %s: %v", name, err)
	}
	return u
}

func threadByID(t *testing.T, db *gorm.DB, publicID string) models.Thread {
	t.Helper()
	var th models.Thread
	if err := db.Where("public_id = ?", publicID).First(&th).Error; err != nil {
		t.Fatalf("load thread %s: %v", publicID, err)
	}
	return th
}

func makeAdmin(t *testing.T, db *gorm.DB, name string) {
	t.Helper()
	if err := db.Model(&models.User{}).Where("username = ?", name).Update("role", "admin").Error; err != nil {
		t.Fatalf("make admin: %v", err)
	}
}

func notificationCount(t *testing.T, db *gorm.DB, userID uint, kind string) int64 {
	t.Helper()
	var n int64
	db.Model(&models.Notification{}).Where("user_id = ? AND kind = ?", userID, kind).Count(&n)
	return n
}

// newPost creates a reply and returns its public ID.
func newPost(t *testing.T, tsURL string, c *http.Client, threadID, body string, parent string) string {
	t.Helper()
	payload := map[string]any{"body": body}
	if parent != "" {
		payload["parent_id"] = parent
	}
	resp := postJSON(t, c, tsURL+"/api/threads/"+threadID+"/posts", payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create post: got %d", resp.StatusCode)
	}
	out := decodeJSON[struct {
		Post struct {
			ID       string  `json:"id"`
			ParentID *string `json:"parent_id"`
		} `json:"post"`
	}](t, resp)
	return out.Post.ID
}

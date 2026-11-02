package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

type postRequest struct {
	Body string `json:"body" binding:"required,min=2,max=8000"`
	// ParentID is the public ID of the post being replied to. One level only.
	ParentID   *string `json:"parent_id"`
	Language   string  `json:"language"`
	EditedByAI bool    `json:"edited_by_ai"`
}

func (a *API) createPost(c *gin.Context) {
	u := auth.Current(c)
	t, err := a.findThread(c.Param("id"))
	if err != nil {
		notFound(c, "That thread does not exist.")
		return
	}
	if t.IsLocked {
		fail(c, http.StatusConflict, "thread_locked",
			"Thread locked. A moderator decided this conversation had peaked.")
		return
	}
	var req postRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Write something. Two characters minimum, which is a low bar.")
		return
	}

	var parentID *uint
	if req.ParentID != nil && *req.ParentID != "" {
		var parent models.Post
		if err := a.db.Where("public_id = ? AND thread_id = ?", *req.ParentID, t.ID).
			First(&parent).Error; err != nil {
			badRequest(c, "The post you're replying to is not in this thread.")
			return
		}
		// Flatten deeper nesting onto the top-level parent: infinite trees are unreadable.
		root := parent.ID
		if parent.ParentID != nil {
			root = *parent.ParentID
		}
		parentID = &root
	}

	now := time.Now()
	p := models.Post{
		PublicID: store.NewPublicID("pst"), ThreadID: t.ID, AuthorID: u.ID, ParentID: parentID,
		Body: strings.TrimSpace(req.Body), Language: normalizeLanguage(req.Language, u.UILanguage),
		EditedByAI: req.EditedByAI,
	}

	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Thread{}).Where("id = ?", t.ID).Updates(map[string]any{
			"post_count":   gorm.Expr("post_count + 1"),
			"last_post_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", u.ID).
			UpdateColumn("post_count", gorm.Expr("post_count + 1")).Error
	})
	if err != nil {
		serverError(c, err)
		return
	}

	a.recordMentions(u, p.Body, t.ID, &p.ID, t.Title)
	// Notify the thread author that someone showed up to their funeral.
	if t.AuthorID != u.ID {
		a.notify(t.AuthorID, "reply",
			"@"+u.Username+" replied to your thread",
			truncate(p.Body, 160), "/t/"+t.PublicID)
	}

	p.Author = u
	c.JSON(http.StatusCreated, gin.H{"post": a.toPostResponse(&p, u, nil)})
}

func (a *API) updatePost(c *gin.Context) {
	u := auth.Current(c)
	var p models.Post
	if err := a.db.Preload("Author").Where("public_id = ?", c.Param("id")).First(&p).Error; err != nil {
		notFound(c, "No such post.")
		return
	}
	if p.AuthorID != u.ID && u.Role != "admin" {
		fail(c, http.StatusForbidden, "forbidden", "Editing other people's grief is not allowed.")
		return
	}
	var req postRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid payload.")
		return
	}
	p.Body = strings.TrimSpace(req.Body)
	p.EditedByAI = req.EditedByAI
	if err := a.db.Save(&p).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"post": a.toPostResponse(&p, u, nil)})
}

func (a *API) deletePost(c *gin.Context) {
	u := auth.Current(c)
	var p models.Post
	if err := a.db.Where("public_id = ?", c.Param("id")).First(&p).Error; err != nil {
		notFound(c, "No such post.")
		return
	}
	if p.AuthorID != u.ID && u.Role != "admin" {
		fail(c, http.StatusForbidden, "forbidden", "Not yours.")
		return
	}
	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&p).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Thread{}).Where("id = ?", p.ThreadID).
			UpdateColumn("post_count", gorm.Expr("post_count - 1")).Error; err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", p.AuthorID).
			UpdateColumn("post_count", gorm.Expr("post_count - 1")).Error
	})
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Post deleted. Plausible deniability restored."})
}

// toPostResponse maps a post. siblings is used to resolve the parent's public ID
// without an extra query when the whole thread is already loaded.
func (a *API) toPostResponse(p *models.Post, viewer *models.User, siblings []models.Post) *postResponse {
	out := &postResponse{
		ID: p.PublicID, Body: p.Body, Language: p.Language,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		IsAnswer: p.IsAnswer, EditedByAI: p.EditedByAI, ReactionCount: p.ReactionCount,
		Author:    toUserResponse(p.Author),
		Reactions: a.reactionSummaries("post", p.ID, viewer),
	}
	if p.ParentID != nil {
		for i := range siblings {
			if siblings[i].ID == *p.ParentID {
				out.ParentID = &siblings[i].PublicID
				break
			}
		}
		if out.ParentID == nil {
			var parent models.Post
			if err := a.db.Select("public_id").Where("id = ?", *p.ParentID).First(&parent).Error; err == nil {
				out.ParentID = &parent.PublicID
			}
		}
	}
	out.CanEdit = viewer != nil && (viewer.ID == p.AuthorID || viewer.Role == "admin")
	return out
}

// recordMentions parses @handles out of a body, persists them, and notifies the targets.
// It ignores self-mentions, because tagging yourself is a cry for help we can't service.
func (a *API) recordMentions(by *models.User, body string, threadID uint, postID *uint, threadTitle string) {
	matches := mentionExpr.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return
	}
	seen := map[string]bool{}
	for _, m := range matches {
		handle := strings.ToLower(m[1])
		if seen[handle] || handle == by.Username {
			continue
		}
		seen[handle] = true

		var target models.User
		if err := a.db.Select("id", "username").Where("username = ?", handle).First(&target).Error; err != nil {
			continue
		}
		_ = a.db.Create(&models.Mention{
			MentionedUserID: target.ID, ByUserID: by.ID, ThreadID: threadID, PostID: postID,
		}).Error

		var thread models.Thread
		link := "/feed"
		if err := a.db.Select("public_id").Where("id = ?", threadID).First(&thread).Error; err == nil {
			link = "/t/" + thread.PublicID
		}
		a.notify(target.ID, "mention",
			"@"+by.Username+" tagged you",
			"In: "+truncate(threadTitle, 120), link)
	}
}

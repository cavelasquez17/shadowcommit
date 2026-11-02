package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

type reactRequest struct {
	Kind string `json:"kind" binding:"required"`
}

// react toggles a reaction on a thread or post. Reacting twice with the same kind
// removes it, which is how every user expects this to work.
func (a *API) react(targetType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := auth.Current(c)
		var req reactRequest
		if err := c.ShouldBindJSON(&req); err != nil || !isValidOption(reactionOptions, req.Kind) {
			badRequest(c, "Unknown reaction. Your feelings are valid but unsupported.")
			return
		}

		targetID, authorID, ok := a.resolveTarget(c, targetType)
		if !ok {
			return
		}

		var existing models.Reaction
		err := a.db.Where("user_id = ? AND kind = ? AND target_type = ? AND target_id = ?",
			u.ID, req.Kind, targetType, targetID).First(&existing).Error

		switch {
		case err == nil:
			if err := a.db.Delete(&existing).Error; err != nil {
				serverError(c, err)
				return
			}
			a.bumpReactionCount(targetType, targetID, -1)
			a.bumpReputation(authorID, -1)
		case errors.Is(err, gorm.ErrRecordNotFound):
			r := models.Reaction{
				UserID: u.ID, Kind: models.ReactionKind(req.Kind),
				TargetType: targetType, TargetID: targetID,
			}
			if err := a.db.Create(&r).Error; err != nil {
				serverError(c, err)
				return
			}
			a.bumpReactionCount(targetType, targetID, 1)
			a.bumpReputation(authorID, 1)
			if authorID != u.ID {
				a.notify(authorID, "reaction",
					"@"+u.Username+" reacted "+emojiFor(req.Kind), "", a.linkFor(targetType, targetID))
			}
		default:
			serverError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"reactions": a.reactionSummaries(targetType, targetID, u)})
	}
}

// resolveTarget maps the :id path param to an internal ID and its author.
func (a *API) resolveTarget(c *gin.Context, targetType string) (targetID, authorID uint, ok bool) {
	switch targetType {
	case "thread":
		var t models.Thread
		if err := a.db.Select("id", "author_id").Where("public_id = ?", c.Param("id")).
			First(&t).Error; err != nil {
			notFound(c, "No such thread.")
			return 0, 0, false
		}
		return t.ID, t.AuthorID, true
	case "post":
		var p models.Post
		if err := a.db.Select("id", "author_id").Where("public_id = ?", c.Param("id")).
			First(&p).Error; err != nil {
			notFound(c, "No such post.")
			return 0, 0, false
		}
		return p.ID, p.AuthorID, true
	}
	badRequest(c, "Unknown target type.")
	return 0, 0, false
}

func (a *API) bumpReactionCount(targetType string, targetID uint, delta int) {
	switch targetType {
	case "thread":
		a.db.Model(&models.Thread{}).Where("id = ?", targetID).
			UpdateColumn("reaction_count", gorm.Expr("reaction_count + ?", delta))
		// Recompute the hot score so the feed reorders immediately.
		var t models.Thread
		if err := a.db.Select("id", "reaction_count", "created_at").
			Where("id = ?", targetID).First(&t).Error; err == nil {
			a.db.Model(&models.Thread{}).Where("id = ?", targetID).
				UpdateColumn("hot_score", hotScore(t.ReactionCount, t.CreatedAt))
		}
	case "post":
		a.db.Model(&models.Post{}).Where("id = ?", targetID).
			UpdateColumn("reaction_count", gorm.Expr("reaction_count + ?", delta))
	}
}

func (a *API) bumpReputation(userID uint, delta int) {
	a.db.Model(&models.User{}).Where("id = ?", userID).
		UpdateColumn("reputation", gorm.Expr("reputation + ?", delta))
}

// reactionSummaries returns every reaction kind with its tally, so the UI can render
// a stable row of buttons instead of a shifting set.
func (a *API) reactionSummaries(targetType string, targetID uint, viewer *models.User) []reactionSummary {
	type row struct {
		Kind  string
		Count int
	}
	var rows []row
	a.db.Model(&models.Reaction{}).
		Select("kind, COUNT(*) as count").
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Group("kind").Scan(&rows)

	counts := make(map[string]int, len(rows))
	for _, r := range rows {
		counts[r.Kind] = r.Count
	}

	mine := map[string]bool{}
	if viewer != nil {
		var kinds []string
		a.db.Model(&models.Reaction{}).
			Where("user_id = ? AND target_type = ? AND target_id = ?", viewer.ID, targetType, targetID).
			Pluck("kind", &kinds)
		for _, k := range kinds {
			mine[k] = true
		}
	}

	out := make([]reactionSummary, 0, len(reactionOptions))
	for _, o := range reactionOptions {
		out = append(out, reactionSummary{
			Kind: o.Value, Emoji: o.Emoji, Label: o.Label,
			Count: counts[o.Value], Mine: mine[o.Value],
		})
	}
	return out
}

func emojiFor(kind string) string {
	for _, o := range reactionOptions {
		if o.Value == kind {
			return o.Emoji
		}
	}
	return "❓"
}

func (a *API) linkFor(targetType string, targetID uint) string {
	switch targetType {
	case "thread":
		var t models.Thread
		if err := a.db.Select("public_id").Where("id = ?", targetID).First(&t).Error; err == nil {
			return "/t/" + t.PublicID
		}
	case "post":
		var p models.Post
		if err := a.db.Select("thread_id").Where("id = ?", targetID).First(&p).Error; err == nil {
			var t models.Thread
			if err := a.db.Select("public_id").Where("id = ?", p.ThreadID).First(&t).Error; err == nil {
				return "/t/" + t.PublicID
			}
		}
	}
	return "/feed"
}

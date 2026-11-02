package handlers

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/ranking"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

// listThreads is the feed. Supports sort=hot|new|top and filters that let you
// browse the apocalypse by nationality, seniority, language, category or tag.
func (a *API) listThreads(c *gin.Context) {
	viewer := auth.Current(c)
	limit, offset := page(c)

	q := a.db.Model(&models.Thread{}).
		Joins("JOIN users ON users.id = threads.author_id AND users.deleted_at IS NULL")

	if v := c.Query("category"); v != "" {
		q = q.Joins("JOIN categories ON categories.id = threads.category_id").
			Where("categories.slug = ?", v)
	}
	if v := c.Query("tag"); v != "" {
		q = q.Joins("JOIN thread_tags ON thread_tags.thread_id = threads.id").
			Joins("JOIN tags ON tags.id = thread_tags.tag_id").Where("tags.slug = ?", v)
	}
	if v := c.Query("country"); v != "" {
		q = q.Where("users.country_code = ?", strings.ToUpper(v))
	}
	if v := c.Query("seniority"); v != "" {
		q = q.Where("users.seniority = ?", v)
	}
	if v := c.Query("cause"); v != "" {
		q = q.Where("users.cause_of_death = ?", v)
	}
	if v := c.Query("language"); v != "" {
		q = q.Where("threads.language = ?", v)
	}
	if v := c.Query("author"); v != "" {
		q = q.Where("users.username = ?", strings.ToLower(v))
	}
	if v := strings.TrimSpace(c.Query("q")); v != "" {
		like := "%" + escapeLike(strings.ToLower(v)) + "%"
		q = q.Where(`(LOWER(threads.title) LIKE ? ESCAPE '\' OR LOWER(threads.body) LIKE ? ESCAPE '\'
			OR EXISTS (SELECT 1 FROM posts WHERE posts.thread_id = threads.id
				AND posts.deleted_at IS NULL AND LOWER(posts.body) LIKE ? ESCAPE '\'))`,
			like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		serverError(c, err)
		return
	}

	order := "threads.is_pinned DESC, threads.hot_score DESC, threads.created_at DESC"
	switch c.DefaultQuery("sort", "hot") {
	case "new":
		order = "threads.is_pinned DESC, threads.created_at DESC"
	case "top":
		order = "threads.is_pinned DESC, threads.reaction_count DESC, threads.post_count DESC"
	case "active":
		order = "threads.is_pinned DESC, threads.last_post_at DESC NULLS LAST, threads.created_at DESC"
	}
	// SQLite understands NULLS LAST only from 3.30; keep it portable by sorting on a computed flag.
	if strings.Contains(order, "NULLS LAST") {
		order = "threads.is_pinned DESC, threads.last_post_at IS NULL, threads.last_post_at DESC"
	}

	var threads []models.Thread
	if err := q.Preload("Author").Preload("Category").Preload("Tags").
		Order(order).Limit(limit).Offset(offset).Find(&threads).Error; err != nil {
		serverError(c, err)
		return
	}

	items := make([]threadResponse, 0, len(threads))
	for i := range threads {
		items = append(items, *a.toThreadResponse(&threads[i], viewer))
	}
	c.JSON(http.StatusOK, paginated[threadResponse]{
		Items: items, Page: offset/limit + 1, Limit: limit, Total: total,
	})
}

// getThread returns one thread with its replies and bumps the view counter.
func (a *API) getThread(c *gin.Context) {
	viewer := auth.Current(c)
	t, err := a.findThread(c.Param("id"))
	if err != nil {
		notFound(c, "That thread is gone. Like the role it described.")
		return
	}
	// Fire-and-forget counter bump: a failed view count is not worth a 500.
	a.db.Model(&models.Thread{}).Where("id = ?", t.ID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1"))
	t.ViewCount++

	var posts []models.Post
	if err := a.db.Preload("Author").Preload("Author.Badges").
		Where("thread_id = ?", t.ID).Order("created_at ASC").Find(&posts).Error; err != nil {
		serverError(c, err)
		return
	}

	postDTOs := make([]postResponse, 0, len(posts))
	for i := range posts {
		postDTOs = append(postDTOs, *a.toPostResponse(&posts[i], viewer, posts))
	}

	c.JSON(http.StatusOK, gin.H{
		"thread": a.toThreadResponse(t, viewer),
		"posts":  postDTOs,
	})
}

type threadRequest struct {
	Title         string   `json:"title" binding:"required,min=8,max=200"`
	Body          string   `json:"body" binding:"required,min=20"`
	Category      string   `json:"category" binding:"required"`
	Language      string   `json:"language"`
	Sarcasm       *int     `json:"sarcasm"`
	Tags          []string `json:"tags"`
	NSFWCorporate bool     `json:"nsfw_corporate"`
}

func (a *API) createThread(c *gin.Context) {
	u := auth.Current(c)
	var req threadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) && len(verrs) > 0 && verrs[0].Field() == "Category" {
			badRequest(c, "Pick a category. Even despair needs filing.")
			return
		}
		badRequest(c, "Title needs 8+ chars and the body needs 20+. Suffering deserves detail.")
		return
	}

	var cat models.Category
	if err := a.db.Where("slug = ?", req.Category).First(&cat).Error; err != nil {
		badRequest(c, "Unknown category: "+req.Category)
		return
	}

	lang := normalizeLanguage(req.Language, u.UILanguage)
	sarcasm := 100
	if req.Sarcasm != nil {
		sarcasm = clamp(*req.Sarcasm, 0, 100)
	}

	t := models.Thread{
		PublicID: store.NewPublicID("thr"), AuthorID: u.ID, CategoryID: cat.ID,
		Title: strings.TrimSpace(req.Title), Slug: slugify(req.Title),
		Body: strings.TrimSpace(req.Body), Language: lang, Sarcasm: sarcasm,
		NSFWCorporate: req.NSFWCorporate,
	}
	t.HotScore = hotScore(0, t.CreatedAt)

	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&t).Error; err != nil {
			return err
		}
		tags, err := resolveTags(tx, req.Tags)
		if err != nil {
			return err
		}
		if len(tags) > 0 {
			if err := tx.Model(&t).Association("Tags").Append(tags); err != nil {
				return err
			}
		}
		if err := tx.Model(&models.User{}).Where("id = ?", u.ID).
			UpdateColumn("thread_count", gorm.Expr("thread_count + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&models.Category{}).Where("id = ?", cat.ID).
			UpdateColumn("thread_count", gorm.Expr("thread_count + 1")).Error
	})
	if err != nil {
		serverError(c, err)
		return
	}

	a.recordMentions(u, t.Body, t.ID, nil, t.Title)

	full, err := a.findThread(t.PublicID)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"thread": a.toThreadResponse(full, u)})
}

func (a *API) updateThread(c *gin.Context) {
	u := auth.Current(c)
	t, err := a.findThread(c.Param("id"))
	if err != nil {
		notFound(c, "No such thread.")
		return
	}
	if t.AuthorID != u.ID && u.Role != "admin" {
		fail(c, http.StatusForbidden, "forbidden", "Not your thread. Rewriting history is management's job.")
		return
	}
	var req threadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid payload.")
		return
	}
	t.Title = strings.TrimSpace(req.Title)
	t.Slug = slugify(req.Title)
	t.Body = strings.TrimSpace(req.Body)
	t.Language = normalizeLanguage(req.Language, t.Language)
	t.NSFWCorporate = req.NSFWCorporate
	if req.Sarcasm != nil {
		t.Sarcasm = clamp(*req.Sarcasm, 0, 100)
	}
	if err := a.db.Save(t).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"thread": a.toThreadResponse(t, u)})
}

func (a *API) deleteThread(c *gin.Context) {
	u := auth.Current(c)
	t, err := a.findThread(c.Param("id"))
	if err != nil {
		notFound(c, "No such thread.")
		return
	}
	if t.AuthorID != u.ID && u.Role != "admin" {
		fail(c, http.StatusForbidden, "forbidden", "Not yours to delete.")
		return
	}
	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(t).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", t.AuthorID).
			UpdateColumn("thread_count", gorm.Expr("thread_count - 1")).Error; err != nil {
			return err
		}
		return tx.Model(&models.Category{}).Where("id = ?", t.CategoryID).
			UpdateColumn("thread_count", gorm.Expr("thread_count - 1")).Error
	})
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted. Unlike your commit history, this is final."})
}

func (a *API) findThread(publicID string) (*models.Thread, error) {
	var t models.Thread
	err := a.db.Preload("Author").Preload("Author.Badges").Preload("Category").Preload("Tags").
		Where("public_id = ?", publicID).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (a *API) toThreadResponse(t *models.Thread, viewer *models.User) *threadResponse {
	out := &threadResponse{
		ID: t.PublicID, Slug: t.Slug, Title: t.Title, Body: t.Body, Language: t.Language,
		Sarcasm: t.Sarcasm, IsPinned: t.IsPinned, IsLocked: t.IsLocked,
		NSFWCorporate: t.NSFWCorporate, ViewCount: t.ViewCount, PostCount: t.PostCount,
		ReactionCount: t.ReactionCount, HotScore: t.HotScore, CreatedAt: t.CreatedAt,
		LastPostAt: t.LastPostAt, Author: toUserResponse(t.Author, viewer),
		Category: toCategoryDTO(t.Category), Tags: toTagDTOs(t.Tags),
		Reactions: a.reactionSummaries("thread", t.ID, viewer),
	}
	out.CanEdit = viewer != nil && (viewer.ID == t.AuthorID || viewer.Role == "admin")
	return out
}

// hotScore delegates to the shared ranking formula using the wall clock.
func hotScore(reactions int, createdAt time.Time) float64 {
	return ranking.HotScore(reactions, createdAt, time.Now())
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func normalizeLanguage(v, fallback string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if isValidOption(languageOptions, v) {
		return v
	}
	if isValidOption(languageOptions, fallback) {
		return fallback
	}
	return "en"
}

// resolveTags upserts tags by slug and bumps their usage counters.
func resolveTags(tx *gorm.DB, labels []string) ([]models.Tag, error) {
	seen := map[string]bool{}
	out := make([]models.Tag, 0, len(labels))
	for _, raw := range labels {
		label := truncate(strings.TrimSpace(raw), 48)
		slug := slugify(label)
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true

		var tag models.Tag
		err := tx.Where("slug = ?", slug).First(&tag).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tag = models.Tag{Slug: slug, Label: label, UsageCount: 1}
			if err := tx.Create(&tag).Error; err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		} else {
			tx.Model(&tag).UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
		}
		out = append(out, tag)
		if len(out) == 5 {
			break // Five tags is plenty. This is a forum, not an SEO farm.
		}
	}
	return out, nil
}

// listCategories and listTags feed the composer's dropdowns.
func (a *API) listCategories(c *gin.Context) {
	var cats []models.Category
	if err := a.db.Order("sort_order ASC, name ASC").Find(&cats).Error; err != nil {
		serverError(c, err)
		return
	}
	out := make([]categoryDTO, 0, len(cats))
	for i := range cats {
		out = append(out, *toCategoryDTO(&cats[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (a *API) listTags(c *gin.Context) {
	var tags []models.Tag
	if err := a.db.Order("usage_count DESC, label ASC").Limit(40).Find(&tags).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toTagDTOs(tags)})
}

// escapeLike makes user input literal inside a LIKE pattern, so searching "100%" or
// "snake_case" does not turn the percent sign or underscore into wildcards.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(s)
}

package handlers

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

// errInviteRequired and errInviteInvalid gate signup when INVITE_ONLY is set. They are
// sentinel errors so callers can tell "no invite" apart from a genuine DB failure.
var (
	errInviteRequired = errors.New("invite code required")
	errInviteInvalid  = errors.New("invite code invalid or expired")
)

// findUsableInvite locks in an unused, unexpired invite code for a new signup. It must
// run inside the same transaction that creates the user, and the caller is responsible
// for marking it used once the user row exists.
func findUsableInvite(tx *gorm.DB, code string) (*models.Invite, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, errInviteRequired
	}
	var inv models.Invite
	if err := tx.Where("code = ?", code).First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errInviteInvalid
		}
		return nil, err
	}
	if inv.UsedByID != nil || time.Now().After(inv.ExpiresAt) {
		return nil, errInviteInvalid
	}
	return &inv, nil
}

// --- Notifications ---

// notify writes a notification. Failures are swallowed on purpose: nobody should get a
// 500 on their reply because the bell icon had a bad day.
func (a *API) notify(userID uint, kind, title, body, link string) {
	_ = a.db.Create(&models.Notification{
		PublicID: store.NewPublicID("ntf"), UserID: userID,
		Kind: kind, Title: truncate(title, 200), Body: truncate(body, 400), Link: link,
	}).Error
}

func (a *API) listNotifications(c *gin.Context) {
	u := auth.Current(c)
	limit, offset := page(c)

	var items []models.Notification
	if err := a.db.Where("user_id = ?", u.ID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		serverError(c, err)
		return
	}
	var unread int64
	a.db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", u.ID, false).Count(&unread)

	c.JSON(http.StatusOK, gin.H{"items": items, "unread": unread})
}

func (a *API) markNotificationsRead(c *gin.Context) {
	u := auth.Current(c)
	if err := a.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", u.ID, false).
		Update("is_read", true).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "All read. Ignorance restored."})
}

// --- Invites ---

const inviteAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newInviteCode() string {
	b := make([]byte, 8)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteAlphabet))))
		if err != nil {
			// Fall back to a time-derived character rather than failing the request.
			b[i] = inviteAlphabet[time.Now().Nanosecond()%len(inviteAlphabet)]
			continue
		}
		b[i] = inviteAlphabet[n.Int64()]
	}
	return string(b)
}

type inviteRequest struct {
	Note string `json:"note" binding:"max=200"`
}

// createInvite mints a code valid for 30 days.
func (a *API) createInvite(c *gin.Context) {
	u := auth.Current(c)
	var req inviteRequest
	_ = c.ShouldBindJSON(&req)

	var active int64
	a.db.Model(&models.Invite{}).
		Where("created_by_id = ? AND used_by_id IS NULL AND expires_at > ?", u.ID, time.Now()).
		Count(&active)
	if active >= 10 {
		fail(c, http.StatusTooManyRequests, "invite_limit",
			"Ten unused invites is enough. Recruit harder, not louder.")
		return
	}

	inv := models.Invite{
		Code: newInviteCode(), CreatedByID: u.ID,
		Note:      truncate(strings.TrimSpace(req.Note), 200),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := a.db.Create(&inv).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"invite": inv,
		"url":    a.cfg.WebBaseURL + "/join?code=" + inv.Code,
	})
}

func (a *API) listInvites(c *gin.Context) {
	u := auth.Current(c)
	var items []models.Invite
	if err := a.db.Where("created_by_id = ?", u.ID).Order("created_at DESC").
		Find(&items).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "base_url": a.cfg.WebBaseURL + "/join?code="})
}

// checkInvite validates a code before the signup flow starts.
func (a *API) checkInvite(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Query("code")))
	var inv models.Invite
	if err := a.db.Where("code = ?", code).First(&inv).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "reason": "That code does not exist."})
		return
	}
	if inv.UsedByID != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "reason": "Already used. Someone beat you to the funeral."})
		return
	}
	if time.Now().After(inv.ExpiresAt) {
		c.JSON(http.StatusOK, gin.H{"valid": false, "reason": "Expired. Like the job posting."})
		return
	}
	var inviter models.User
	a.db.Select("username", "display_name", "avatar_url").Where("id = ?", inv.CreatedByID).First(&inviter)
	c.JSON(http.StatusOK, gin.H{
		"valid": true, "note": inv.Note,
		"inviter": gin.H{
			"username": inviter.Username, "display_name": inviter.DisplayName,
			"avatar_url": inviter.AvatarURL,
		},
	})
}

// --- Mentions ---

// listMentions shows every time someone dragged you into their thread.
func (a *API) listMentions(c *gin.Context) {
	u := auth.Current(c)
	limit, offset := page(c)

	type mentionRow struct {
		CreatedAt   time.Time `json:"created_at"`
		ThreadID    string    `json:"thread_id"`
		ThreadTitle string    `json:"thread_title"`
		ByUsername  string    `json:"by_username"`
		ByAvatar    string    `json:"by_avatar"`
	}
	var rows []mentionRow
	err := a.db.Model(&models.Mention{}).
		Select(`mentions.created_at, threads.public_id as thread_id, threads.title as thread_title,
		        users.username as by_username, users.avatar_url as by_avatar`).
		Joins("JOIN threads ON threads.id = mentions.thread_id").
		Joins("JOIN users ON users.id = mentions.by_user_id").
		Where("mentions.mentioned_user_id = ?", u.ID).
		Order("mentions.created_at DESC").Limit(limit).Offset(offset).Scan(&rows).Error
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

// searchUsers backs the @mention autocomplete in the composer.
func (a *API) searchUsers(c *gin.Context) {
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	if len(q) < 1 {
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	var users []models.User
	a.db.Select("public_id", "username", "display_name", "avatar_url", "country_code", "seniority").
		Where("LOWER(username) LIKE ? OR LOWER(display_name) LIKE ?", q+"%", "%"+q+"%").
		Order("reputation DESC").Limit(8).Find(&users)

	items := make([]gin.H, 0, len(users))
	for _, u := range users {
		items = append(items, gin.H{
			"username": u.Username, "display_name": u.DisplayName,
			"avatar_url": u.AvatarURL, "country_code": u.CountryCode,
			"seniority": u.Seniority,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

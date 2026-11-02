package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

const (
	oauthStateCookie  = "sc_oauth_state"
	oauthInviteCookie = "sc_oauth_invite"
)

// githubStart redirects the browser into GitHub's consent screen.
func (a *API) githubStart(c *gin.Context) {
	if a.github == nil {
		fail(c, http.StatusServiceUnavailable, "github_disabled",
			"GitHub login is not configured. Set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET.")
		return
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		serverError(c, err)
		return
	}
	state := hex.EncodeToString(b)
	// The state cookie is short-lived and only needs to survive the round trip.
	http.SetCookie(c.Writer, &http.Cookie{
		Name: oauthStateCookie, Value: state, Path: "/", MaxAge: 600,
		HttpOnly: true, Secure: a.iss.Secure(), SameSite: http.SameSiteLaxMode,
	})
	// An invite code, if the caller arrived via a /join link, rides along in its own
	// cookie so it survives the redirect to GitHub and back.
	if invite := strings.TrimSpace(c.Query("invite")); invite != "" {
		http.SetCookie(c.Writer, &http.Cookie{
			Name: oauthInviteCookie, Value: invite, Path: "/", MaxAge: 600,
			HttpOnly: true, Secure: a.iss.Secure(), SameSite: http.SameSiteLaxMode,
		})
	}
	c.Redirect(http.StatusTemporaryRedirect, a.github.AuthCodeURL(state))
}

func clearOAuthCookies(c *gin.Context, secure bool) {
	for _, name := range []string{oauthStateCookie, oauthInviteCookie} {
		http.SetCookie(c.Writer, &http.Cookie{
			Name: name, Value: "", Path: "/", MaxAge: -1,
			HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
		})
	}
}

// githubCallback completes the OAuth dance, upserts the user, and bounces to the web app.
func (a *API) githubCallback(c *gin.Context) {
	if a.github == nil {
		fail(c, http.StatusServiceUnavailable, "github_disabled", "GitHub login is not configured.")
		return
	}
	want, err := c.Cookie(oauthStateCookie)
	if err != nil || want == "" || c.Query("state") != want {
		fail(c, http.StatusBadRequest, "bad_state", "OAuth state mismatch. Someone is being clever.")
		return
	}
	inviteCode, _ := c.Cookie(oauthInviteCookie)
	clearOAuthCookies(c, a.iss.Secure())

	ghUser, ghRepos, err := a.github.Exchange(c.Request.Context(), c.Query("code"))
	if err != nil {
		serverError(c, err)
		return
	}

	u, isNew, err := a.upsertGitHubUser(ghUser, ghRepos, inviteCode)
	if err != nil {
		switch {
		case errors.Is(err, errInviteRequired), errors.Is(err, errInviteInvalid):
			fail(c, http.StatusForbidden, "invite_required",
				"This funeral is invite-only. Bring a code.")
		default:
			serverError(c, err)
		}
		return
	}
	token, err := a.iss.Token(u)
	if err != nil {
		serverError(c, err)
		return
	}
	a.iss.SetCookie(c, token)

	dest := a.cfg.WebBaseURL + "/feed"
	if isNew {
		// New arrivals get routed through the onboarding wizard to declare their cause of death.
		dest = a.cfg.WebBaseURL + "/onboarding"
	}
	c.Redirect(http.StatusTemporaryRedirect, dest)
}

// upsertGitHubUser logs in a returning GitHub identity or, for a first-time arrival,
// creates one. New-user creation consumes an invite atomically when INVITE_ONLY is on;
// returning users never need one — the gate is for registration, not re-entry.
func (a *API) upsertGitHubUser(gh *auth.GHUser, repos []auth.GHRepo, inviteCode string) (*models.User, bool, error) {
	var u models.User
	err := a.db.Where("git_hub_id = ?", gh.ID).First(&u).Error
	if err == nil {
		// Returning user: refresh the fields GitHub owns, leave their confessions alone.
		u.Username = gh.Login
		u.AvatarURL = gh.AvatarURL
		u.GitHubURL = gh.HTMLURL
		if u.DisplayName == "" {
			u.DisplayName = firstNonEmpty(gh.Name, gh.Login)
		}
		return &u, false, a.db.Save(&u).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	ghID := gh.ID
	u = models.User{
		PublicID:        store.NewPublicID("usr"),
		GitHubID:        &ghID,
		Username:        a.uniqueUsername(gh.Login),
		Email:           gh.Email,
		AvatarURL:       gh.AvatarURL,
		DisplayName:     firstNonEmpty(gh.Name, gh.Login),
		Bio:             truncate(gh.Bio, 500),
		GitHubURL:       gh.HTMLURL,
		WebsiteURL:      truncate(gh.Blog, 255),
		Seniority:       models.SenioritySenior,
		CauseOfDeath:    models.CauseStillAlive,
		UILanguage:      "en",
		SpokenLanguages: "en",
		IsEmployed:      true,
		Role:            "member",
	}
	txErr := a.db.Transaction(func(tx *gorm.DB) error {
		var inv *models.Invite
		if a.cfg.InviteOnly {
			var ierr error
			inv, ierr = findUsableInvite(tx, inviteCode)
			if ierr != nil {
				return ierr
			}
		}
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		if inv != nil {
			return tx.Model(inv).Updates(map[string]any{"used_by_id": u.ID, "used_at": time.Now()}).Error
		}
		return nil
	})
	if txErr != nil {
		return nil, false, txErr
	}

	// Import their public repos so the profile is not empty on arrival. Best-effort:
	// a failed repo import should never block the signup that already committed.
	for _, r := range repos {
		if r.Fork {
			continue
		}
		_ = a.db.Create(&models.Repo{
			PublicID:    store.NewPublicID("rep"),
			UserID:      u.ID,
			Name:        truncate(r.Name, 140),
			URL:         r.HTMLURL,
			Description: truncate(r.Description, 300),
			Language:    truncate(r.Language, 40),
			Stars:       r.Stars,
		}).Error
	}
	return &u, true, nil
}

type devLoginRequest struct {
	Username   string `json:"username" binding:"required,min=2,max=39"`
	InviteCode string `json:"invite_code"`
}

// devLogin creates or re-enters a dev-only local persona with no password at all.
// It is loudly gated on DEV_LOGIN and refuses to exist in production. It will never
// authenticate into an account that has a real GitHubID — that identity belongs to
// whoever owns that GitHub account, not to whoever typed the username.
func (a *API) devLogin(c *gin.Context) {
	if !a.cfg.DevLoginEnabled {
		fail(c, http.StatusForbidden, "dev_login_disabled",
			"Dev login is off. Use GitHub like a person with boundaries.")
		return
	}
	var req devLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Give me a username. Any username. That is the entire requirement.")
		return
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))

	var u models.User
	err := a.db.Where("username = ?", username).First(&u).Error
	isNew := errors.Is(err, gorm.ErrRecordNotFound)
	switch {
	case err == nil:
		if u.GitHubID != nil {
			fail(c, http.StatusForbidden, "dev_login_taken",
				"That handle belongs to a real GitHub identity. Log in with GitHub, impostor.")
			return
		}
	case isNew:
		u = models.User{
			PublicID:        store.NewPublicID("usr"),
			Username:        username,
			DisplayName:     username,
			AvatarURL:       "https://api.dicebear.com/9.x/bottts-neutral/svg?seed=" + username,
			Seniority:       models.SenioritySenior,
			CauseOfDeath:    models.CauseStillAlive,
			UILanguage:      "en",
			SpokenLanguages: "en",
			IsEmployed:      true,
			Role:            "member",
		}
		txErr := a.db.Transaction(func(tx *gorm.DB) error {
			var inv *models.Invite
			if a.cfg.InviteOnly {
				var ierr error
				inv, ierr = findUsableInvite(tx, req.InviteCode)
				if ierr != nil {
					return ierr
				}
			}
			if err := tx.Create(&u).Error; err != nil {
				return err
			}
			if inv != nil {
				return tx.Model(inv).Updates(map[string]any{"used_by_id": u.ID, "used_at": time.Now()}).Error
			}
			return nil
		})
		if txErr != nil {
			switch {
			case errors.Is(txErr, errInviteRequired), errors.Is(txErr, errInviteInvalid):
				fail(c, http.StatusForbidden, "invite_required", "This funeral is invite-only. Bring a code.")
			default:
				serverError(c, txErr)
			}
			return
		}
	default:
		serverError(c, err)
		return
	}

	token, err := a.iss.Token(&u)
	if err != nil {
		serverError(c, err)
		return
	}
	a.iss.SetCookie(c, token)
	c.JSON(http.StatusOK, gin.H{"user": toUserResponse(&u, nil), "is_new": isNew})
}

// logout clears the session. Unlike a layoff, this one is reversible.
func (a *API) logout(c *gin.Context) {
	a.iss.ClearCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out. Your desk has already been reassigned."})
}

// me returns the authenticated user, or 401.
func (a *API) me(c *gin.Context) {
	u := auth.Current(c)
	if u == nil {
		fail(c, http.StatusUnauthorized, "unauthorized", "Not logged in.")
		return
	}
	var unread int64
	a.db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", u.ID, false).Count(&unread)
	c.JSON(http.StatusOK, gin.H{"user": toUserResponse(u, nil), "unread_notifications": unread})
}

// uniqueUsername appends a numeric suffix until the handle is free.
func (a *API) uniqueUsername(base string) string {
	base = strings.ToLower(strings.TrimSpace(base))
	if base == "" {
		base = "anon"
	}
	candidate := base
	for i := 2; i < 1000; i++ {
		var count int64
		a.db.Model(&models.User{}).Where("username = ?", candidate).Count(&count)
		if count == 0 {
			return candidate
		}
		candidate = base + "-" + itoa(i)
	}
	return base + "-" + hex.EncodeToString([]byte(time.Now().Format("150405")))
}

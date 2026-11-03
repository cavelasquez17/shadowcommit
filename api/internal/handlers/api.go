// Package handlers exposes the HTTP surface of ShadowCommit.
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
	"github.com/cavelasquez17/shadowcommit/api/internal/config"
)

// API bundles the dependencies every handler needs.
type API struct {
	db     *gorm.DB
	cfg    *config.Config
	iss    *auth.Issuer
	github *auth.GitHub

	authLimiter  *rateLimiter
	writeLimiter *rateLimiter
}

// New builds the API.
func New(db *gorm.DB, cfg *config.Config, iss *auth.Issuer, gh *auth.GitHub) *API {
	return &API{
		db: db, cfg: cfg, iss: iss, github: gh,
		// Auth endpoints: tight budget, they're the ones worth brute-forcing.
		authLimiter: newRateLimiter(10, time.Minute),
		// Content creation: looser, still enough to stop a spam script.
		writeLimiter: newRateLimiter(20, time.Minute),
	}
}

// Close stops the background goroutines the API started. Call it on shutdown.
func (a *API) Close() {
	a.authLimiter.Stop()
	a.writeLimiter.Stop()
}

// fail writes a consistent error envelope. The message is user-facing and in character.
func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": code, "message": message})
}

func badRequest(c *gin.Context, message string) {
	fail(c, http.StatusBadRequest, "bad_request", message)
}
func notFound(c *gin.Context, message string) { fail(c, http.StatusNotFound, "not_found", message) }
func serverError(c *gin.Context, err error) {
	ref := errRef()
	log.Printf("[error %s] %s %s: %v", ref, c.Request.Method, c.Request.URL.Path, err)
	fail(c, http.StatusInternalServerError, "internal_error",
		"The server broke. Ironically, no AI was involved. Reference: "+ref)
}

// errRef is a short correlation id linking a client-facing 500 to the real error in the
// server log, without handing the client the raw error text (schema names, driver
// messages, file paths).
func errRef() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "xxxxxxxx"
	}
	return hex.EncodeToString(b)
}

// page reads pagination params, clamped so nobody can ask for the whole graveyard at once.
func page(c *gin.Context) (limit, offset int) {
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if p < 1 {
		p = 1
	}
	return limit, (p - 1) * limit
}

var (
	nonSlug     = regexp.MustCompile(`[^a-z0-9]+`)
	mentionExpr = regexp.MustCompile(`@([A-Za-z0-9][A-Za-z0-9-]{0,38})`)
)

// slugify produces a URL fragment. Non-latin scripts collapse to empty, and that is fine —
// the public ID is what actually routes.
func slugify(s string) string {
	var b strings.Builder
	// Decompose accents (ó -> o + combining mark) and drop the marks, so "qué" becomes
	// "que" instead of losing the letter entirely.
	decomposed := norm.NFD.String(strings.ToLower(s))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		switch {
		case unicode.IsLetter(r) && r < unicode.MaxASCII, unicode.IsDigit(r) && r < unicode.MaxASCII:
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(nonSlug.ReplaceAllString(b.String(), "-"), "-")
	if len(out) > 80 {
		out = strings.Trim(out[:80], "-")
	}
	return out
}

// meta serves the enum vocabulary so the frontend never hardcodes it.
func (a *API) meta(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"seniorities":     seniorityOptions,
		"causes_of_death": causeOptions,
		"reactions":       reactionOptions,
		"countries":       countryOptions,
		"languages":       languageOptions,
		"invite_only":     a.cfg.InviteOnly,
		"dev_login":       a.cfg.DevLoginEnabled,
		"github_enabled":  a.github != nil,
	})
}

// orEmpty returns s, or an empty non-nil slice when s is nil. A nil slice marshals to JSON null,
// but the API promises arrays (and the web app calls .map on them), so an empty table must be [].
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

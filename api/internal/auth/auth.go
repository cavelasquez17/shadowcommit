// Package auth issues and verifies sessions, and talks to GitHub's OAuth.
package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

const (
	// SessionCookie is the name of the cookie holding the JWT.
	SessionCookie = "sc_session"
	// ContextUserKey is where the authenticated user lands on the gin context.
	ContextUserKey = "current_user"

	sessionTTL = 30 * 24 * time.Hour
)

// ErrUnauthorized is returned when the caller is nobody.
var ErrUnauthorized = errors.New("unauthorized")

// Claims is the JWT payload. Deliberately minimal: the DB is the source of truth.
type Claims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
}

// Issuer mints and validates session tokens.
type Issuer struct {
	secret []byte
	secure bool
}

// NewIssuer builds an Issuer from config.
func NewIssuer(cfg *config.Config) *Issuer {
	return &Issuer{
		secret: []byte(cfg.JWTSecret),
		secure: strings.HasPrefix(cfg.AppBaseURL, "https://"),
	}
}

// Token returns a signed JWT for the user.
func (i *Issuer) Token(u *models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.PublicID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(sessionTTL)),
			Issuer:    "shadowcommit",
		},
		Username: u.Username,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
}

// Parse validates a token string and returns its claims.
func (i *Issuer) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return i.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, ErrUnauthorized
	}
	return claims, nil
}

// SetCookie writes the session cookie on the response.
func (i *Issuer) SetCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   i.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie logs the user out, which is the second most common form of exit here.
func (i *Issuer) ClearCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   i.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Secure reports whether cookies should carry the Secure attribute, derived from
// whether AppBaseURL is HTTPS. Handlers outside this package (OAuth state/invite
// cookies) use this so every cookie in the app follows the same rule.
func (i *Issuer) Secure() bool {
	return i.secure
}

func tokenFromRequest(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if ck, err := c.Cookie(SessionCookie); err == nil {
		return ck
	}
	return ""
}

// Optional resolves the current user if a valid session exists, and moves on otherwise.
func Optional(db *gorm.DB, iss *Issuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if u := resolve(c, db, iss); u != nil {
			c.Set(ContextUserKey, u)
		}
		c.Next()
	}
}

// Required aborts with 401 unless a valid session resolves to a live user.
func Required(db *gorm.DB, iss *Issuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := resolve(c, db, iss)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "You are not logged in. Much like your job, this session does not exist.",
			})
			return
		}
		c.Set(ContextUserKey, u)
		c.Next()
	}
}

// AdminRequired gates moderation endpoints.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := Current(c)
		if u == nil || u.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Nice try. Escalating privileges is what got the AI in trouble too.",
			})
			return
		}
		c.Next()
	}
}

func resolve(c *gin.Context, db *gorm.DB, iss *Issuer) *models.User {
	tok := tokenFromRequest(c)
	if tok == "" {
		return nil
	}
	claims, err := iss.Parse(tok)
	if err != nil {
		return nil
	}
	var u models.User
	if err := db.Preload("Repos").Preload("Badges").
		Where("public_id = ?", claims.Subject).First(&u).Error; err != nil {
		return nil
	}
	return &u
}

// Current returns the authenticated user, or nil.
func Current(c *gin.Context) *models.User {
	v, ok := c.Get(ContextUserKey)
	if !ok {
		return nil
	}
	u, ok := v.(*models.User)
	if !ok {
		return nil
	}
	return u
}

// HashPassword hashes a password for the email/password fallback path.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword compares a plaintext password against a stored hash.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

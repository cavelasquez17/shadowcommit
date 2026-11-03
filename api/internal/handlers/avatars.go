package handlers

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/cavelasquez17/shadowcommit/api/internal/avatar"
)

// usernamePattern matches the handles people can register. Anything else is refused rather than
// drawn, so the endpoint cannot be used to fabricate arbitrary responses.
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,38}$`)

// avatarOf serves a member's default avatar. The image is deterministic, so it is safe to cache.
func (a *API) avatarOf(c *gin.Context) {
	name := c.Param("username")
	if !usernamePattern.MatchString(name) {
		badRequest(c, "That is not a handle anyone could have registered.")
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/svg+xml", avatar.SVG(name))
}

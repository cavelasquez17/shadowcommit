package handlers

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
)

// Router builds the full HTTP router. Public routes use AuthOptional so responses can
// include "did I already react to this" without forcing a login.
func (a *API) Router() *gin.Engine {
	if a.cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     a.cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/healthz", func(c *gin.Context) {
		version := a.cfg.Version
		if version == "" {
			version = "dev"
		}
		c.JSON(http.StatusOK, gin.H{"status": "alive", "unlike": "your role", "version": version})
	})

	api := r.Group("/api")
	optional := auth.Optional(a.db, a.iss)
	required := auth.Required(a.db, a.iss)

	// --- Auth ---
	authGroup := api.Group("/auth")
	authLimit := a.authLimiter.middleware()
	{
		authGroup.GET("/github", authLimit, a.githubStart)
		authGroup.GET("/github/callback", authLimit, a.githubCallback)
		authGroup.POST("/dev-login", authLimit, a.devLogin)
		authGroup.POST("/logout", a.logout)
		authGroup.GET("/me", optional, a.me)
	}

	// --- Reference data ---
	api.GET("/meta", a.meta)
	api.GET("/categories", a.listCategories)
	api.GET("/tags", a.listTags)
	api.GET("/stats", a.stats)
	api.GET("/invites/check", a.checkInvite)

	// --- Threads ---
	writeLimit := a.writeLimiter.middleware()
	api.GET("/threads", optional, a.listThreads)
	api.GET("/threads/:id", optional, a.getThread)
	api.POST("/threads", required, writeLimit, a.createThread)
	api.PATCH("/threads/:id", required, a.updateThread)
	api.DELETE("/threads/:id", required, a.deleteThread)
	api.POST("/threads/:id/react", required, a.react("thread"))
	api.POST("/threads/:id/posts", required, writeLimit, a.createPost)

	// --- Posts ---
	api.PATCH("/posts/:id", required, a.updatePost)
	api.DELETE("/posts/:id", required, a.deletePost)
	api.POST("/posts/:id/react", required, a.react("post"))

	// --- Users ---
	api.GET("/users", optional, a.listUsers)
	api.GET("/users/search", a.searchUsers)
	api.GET("/users/:username", optional, a.getUser)
	api.POST("/users/:username/follow", required, a.toggleFollow)
	api.PATCH("/profile", required, a.updateProfile)
	api.POST("/profile/repos", required, a.addRepo)
	api.DELETE("/profile/repos/:id", required, a.deleteRepo)

	// --- Social plumbing ---
	api.GET("/notifications", required, a.listNotifications)
	api.POST("/notifications/read", required, a.markNotificationsRead)
	api.GET("/mentions", required, a.listMentions)
	api.GET("/invites", required, a.listInvites)
	api.POST("/invites", required, a.createInvite)

	// --- Jobs: "Hire the Fallen" ---
	api.GET("/jobs", optional, a.listJobs)
	api.GET("/jobs/stats", a.jobStats)
	api.GET("/jobs/:id", optional, a.getJob)
	api.POST("/jobs", required, writeLimit, a.createJob)
	api.POST("/jobs/:id/sponsor", required, a.toggleSponsored)
	api.POST("/jobs/:id/filled", required, a.markJobFilled)
	api.DELETE("/jobs/:id", required, a.deleteJob)

	apiNotFound := func(c *gin.Context) {
		fail(c, http.StatusNotFound, "not_found",
			"That endpoint does not exist. It was deprecated by an intern with an LLM.")
	}
	if a.cfg.StaticDir != "" {
		r.NoRoute(staticHandler(a.cfg.StaticDir, apiNotFound))
	} else {
		r.NoRoute(apiNotFound)
	}
	return r
}

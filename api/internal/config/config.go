// Package config loads runtime configuration from the environment.
package config

import (
	"log"
	"os"
	"strings"
)

// Config is everything the API needs to boot.
type Config struct {
	Port        string
	Env         string
	DBDriver    string // "sqlite" or "postgres"
	DSN         string
	JWTSecret   string
	AppBaseURL  string
	WebBaseURL  string
	CORSOrigins []string

	GitHubClientID     string
	GitHubClientSecret string

	// DevLoginEnabled lets you become anyone without a GitHub app. Never enable in prod.
	DevLoginEnabled bool
	// InviteOnly gates registration behind an invite code.
	InviteOnly bool
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return fallback
}

// Load reads config from the environment, applying local-friendly defaults.
func Load() *Config {
	c := &Config{
		Port:               env("PORT", "8080"),
		Env:                env("APP_ENV", "development"),
		DBDriver:           env("DB_DRIVER", "sqlite"),
		DSN:                env("DATABASE_URL", "shadowcommit.db"),
		JWTSecret:          env("JWT_SECRET", "dev-secret-rotate-me-or-get-fired"),
		AppBaseURL:         env("APP_BASE_URL", "http://localhost:8080"),
		WebBaseURL:         env("WEB_BASE_URL", "http://localhost:5173"),
		GitHubClientID:     env("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: env("GITHUB_CLIENT_SECRET", ""),
	}
	c.CORSOrigins = strings.Split(env("CORS_ORIGINS", c.WebBaseURL+",http://localhost:4173"), ",")
	// DevLoginEnabled defaults to false unconditionally — it must be turned on explicitly via
	// DEV_LOGIN=true (as .env.example does for local dev). Tying the default to APP_ENV meant
	// any deployment that simply forgot to set APP_ENV="production" booted with account-takeover
	// dev-login wide open by default (see AGENT.md security notes).
	c.DevLoginEnabled = envBool("DEV_LOGIN", false)
	c.InviteOnly = envBool("INVITE_ONLY", false)

	if c.Env == "production" {
		if c.JWTSecret == "dev-secret-rotate-me-or-get-fired" {
			log.Fatal("refusing to boot production with the default JWT_SECRET")
		}
		if c.DevLoginEnabled {
			log.Fatal("DEV_LOGIN must be off in production")
		}
	}
	return c
}

// IsGitHubConfigured reports whether OAuth can actually work.
func (c *Config) IsGitHubConfigured() bool {
	return c.GitHubClientID != "" && c.GitHubClientSecret != ""
}

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	ghoauth "golang.org/x/oauth2/github"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
)

// GitHub wraps the OAuth2 dance plus the two API calls we care about.
type GitHub struct {
	conf *oauth2.Config
	http *http.Client
}

// NewGitHub builds a GitHub OAuth client. Returns nil when credentials are absent.
func NewGitHub(cfg *config.Config) *GitHub {
	if !cfg.IsGitHubConfigured() {
		return nil
	}
	return &GitHub{
		conf: &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			Endpoint:     ghoauth.Endpoint,
			// Must match the callback URL registered with the GitHub OAuth app, which lives
			// on the API's own origin (see .env.example) — not the web app's origin.
			RedirectURL: cfg.AppBaseURL + "/api/auth/github/callback",
			Scopes:      []string{"read:user", "user:email"},
		},
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// AuthCodeURL returns the URL to redirect the browser to.
func (g *GitHub) AuthCodeURL(state string) string {
	return g.conf.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// GHUser is the subset of GitHub's user payload we store.
type GHUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Bio       string `json:"bio"`
	Blog      string `json:"blog"`
	Location  string `json:"location"`
	Email     string `json:"email"`
}

// GHRepo is the subset of a repository we surface on a profile.
type GHRepo struct {
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stargazers_count"`
	Fork        bool   `json:"fork"`
}

// Exchange trades the callback code for a user profile and their top repos.
func (g *GitHub) Exchange(ctx context.Context, code string) (*GHUser, []GHRepo, error) {
	tok, err := g.conf.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("exchange code: %w", err)
	}
	client := g.conf.Client(ctx, tok)

	var user GHUser
	if err := getJSON(ctx, client, "https://api.github.com/user", &user); err != nil {
		return nil, nil, err
	}
	if user.Email == "" {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err == nil {
			for _, e := range emails {
				if e.Primary && e.Verified {
					user.Email = e.Email
					break
				}
			}
		}
	}

	var repos []GHRepo
	// Best effort: a profile without repos is sad but not fatal.
	_ = getJSON(ctx, client, "https://api.github.com/user/repos?sort=updated&per_page=12", &repos)

	return &user, repos, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("call %s: unexpected status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

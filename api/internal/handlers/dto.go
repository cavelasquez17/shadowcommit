package handlers

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

// userResponse is the public shape of a user. No emails, no password hashes, no regrets.
type userResponse struct {
	ID              string     `json:"id"`
	Username        string     `json:"username"`
	DisplayName     string     `json:"display_name"`
	AvatarURL       string     `json:"avatar_url"`
	Bio             string     `json:"bio"`
	CountryCode     string     `json:"country_code"`
	Seniority       string     `json:"seniority"`
	YearsExp        int        `json:"years_exp"`
	CauseOfDeath    string     `json:"cause_of_death"`
	Epitaph         string     `json:"epitaph"`
	CurrentRole     string     `json:"current_role"`
	IsEmployed      bool       `json:"is_employed"`
	FiredAt         *time.Time `json:"fired_at"`
	GitHubURL       string     `json:"github_url"`
	WebsiteURL      string     `json:"website_url"`
	LinkedInURL     string     `json:"linkedin_url"`
	UILanguage      string     `json:"ui_language"`
	SpokenLanguages []string   `json:"spoken_languages"`
	CopiumMode      bool       `json:"copium_mode"`
	ThreadCount     int        `json:"thread_count"`
	PostCount       int        `json:"post_count"`
	Reputation      int        `json:"reputation"`
	Role            string     `json:"role"`
	CreatedAt       time.Time  `json:"created_at"`
	Repos           []repoDTO  `json:"repos"`
	Badges          []badgeDTO `json:"badges"`
	// DaysSinceFired powers the "days survived" counter on the profile.
	DaysSinceFired *int `json:"days_since_fired"`
	// IsFollowing is set when the request is authenticated.
	IsFollowing   *bool `json:"is_following,omitempty"`
	FollowerCount int   `json:"follower_count"`
}

type repoDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stars"`
	WrittenByAI bool   `json:"written_by_ai"`
}

type badgeDTO struct {
	Slug        string `json:"slug"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
	Color       string `json:"color"`
}

type categoryDTO struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
	Color       string `json:"color"`
	ThreadCount int    `json:"thread_count"`
}

type tagDTO struct {
	Slug       string `json:"slug"`
	Label      string `json:"label"`
	UsageCount int    `json:"usage_count"`
}

// reactionSummary is the per-kind tally plus whether the caller already reacted.
type reactionSummary struct {
	Kind  string `json:"kind"`
	Emoji string `json:"emoji"`
	Label string `json:"label"`
	Count int    `json:"count"`
	Mine  bool   `json:"mine"`
}

type threadResponse struct {
	ID            string            `json:"id"`
	Slug          string            `json:"slug"`
	Title         string            `json:"title"`
	Body          string            `json:"body"`
	Language      string            `json:"language"`
	Sarcasm       int               `json:"sarcasm"`
	IsPinned      bool              `json:"is_pinned"`
	IsLocked      bool              `json:"is_locked"`
	NSFWCorporate bool              `json:"nsfw_corporate"`
	ViewCount     int               `json:"view_count"`
	PostCount     int               `json:"post_count"`
	ReactionCount int               `json:"reaction_count"`
	HotScore      float64           `json:"hot_score"`
	CreatedAt     time.Time         `json:"created_at"`
	LastPostAt    *time.Time        `json:"last_post_at"`
	Author        *userResponse     `json:"author"`
	Category      *categoryDTO      `json:"category"`
	Tags          []tagDTO          `json:"tags"`
	Reactions     []reactionSummary `json:"reactions"`
	CanEdit       bool              `json:"can_edit"`
}

type postResponse struct {
	ID            string            `json:"id"`
	Body          string            `json:"body"`
	Language      string            `json:"language"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	ParentID      *string           `json:"parent_id"`
	IsAnswer      bool              `json:"is_answer"`
	EditedByAI    bool              `json:"edited_by_ai"`
	ReactionCount int               `json:"reaction_count"`
	Author        *userResponse     `json:"author"`
	Reactions     []reactionSummary `json:"reactions"`
	CanEdit       bool              `json:"can_edit"`
}

type paginated[T any] struct {
	Items []T   `json:"items"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// toUserResponse maps a user. viewer may be nil for anonymous requests.
func toUserResponse(u *models.User, viewer *models.User) *userResponse {
	if u == nil {
		return nil
	}
	out := &userResponse{
		ID: u.PublicID, Username: u.Username, DisplayName: u.DisplayName,
		AvatarURL: u.AvatarURL, Bio: u.Bio, CountryCode: u.CountryCode,
		Seniority: string(u.Seniority), YearsExp: u.YearsExp,
		CauseOfDeath: string(u.CauseOfDeath), Epitaph: u.Epitaph,
		CurrentRole: u.CurrentRole, IsEmployed: u.IsEmployed, FiredAt: u.FiredAt,
		GitHubURL: u.GitHubURL, WebsiteURL: u.WebsiteURL, LinkedInURL: u.LinkedInURL,
		UILanguage: u.UILanguage, CopiumMode: u.CopiumMode,
		ThreadCount: u.ThreadCount, PostCount: u.PostCount, Reputation: u.Reputation,
		Role: u.Role, CreatedAt: u.CreatedAt,
		SpokenLanguages: splitCSV(u.SpokenLanguages),
		Repos:           make([]repoDTO, 0, len(u.Repos)),
		Badges:          make([]badgeDTO, 0, len(u.Badges)),
	}
	if u.FiredAt != nil {
		d := int(time.Since(*u.FiredAt).Hours() / 24)
		out.DaysSinceFired = &d
	}
	for _, r := range u.Repos {
		out.Repos = append(out.Repos, repoDTO{
			ID: r.PublicID, Name: r.Name, URL: r.URL, Description: r.Description,
			Language: r.Language, Stars: r.Stars, WrittenByAI: r.WrittenByAI,
		})
	}
	for _, b := range u.Badges {
		out.Badges = append(out.Badges, badgeDTO{
			Slug: b.Slug, Label: b.Label, Description: b.Description, Emoji: b.Emoji, Color: b.Color,
		})
	}
	return out
}

func toCategoryDTO(c *models.Category) *categoryDTO {
	if c == nil {
		return nil
	}
	return &categoryDTO{
		Slug: c.Slug, Name: c.Name, Description: c.Description,
		Emoji: c.Emoji, Color: c.Color, ThreadCount: c.ThreadCount,
	}
}

func toTagDTOs(tags []models.Tag) []tagDTO {
	out := make([]tagDTO, 0, len(tags))
	for _, t := range tags {
		out = append(out, tagDTO{Slug: t.Slug, Label: t.Label, UsageCount: t.UsageCount})
	}
	return out
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func itoa(i int) string { return strconv.Itoa(i) }

// isHTTPURL reports whether s is an absolute http(s) URL with a host. Used to keep
// profile/job/repo links from carrying a javascript: or data: payload.
func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// jobPostingResponse is the public shape of a job listing.
type jobPostingResponse struct {
	ID          string     `json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	Company     string     `json:"company"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	ApplyURL    string     `json:"apply_url"`
	Location    string     `json:"location"`
	Remote      bool       `json:"remote"`
	CountryCode string     `json:"country_code"`
	SalaryMin   *int       `json:"salary_min"`
	SalaryMax   *int       `json:"salary_max"`
	Currency    string     `json:"currency"`
	Seniority   string     `json:"seniority"`
	Language    string     `json:"language"`
	IsSponsored bool       `json:"is_sponsored"`
	IsFilled    bool       `json:"is_filled"`
	ViewCount   int        `json:"view_count"`
	PostedBy    *userResponse `json:"posted_by,omitempty"`
}

func toJobPostingResponse(j *models.JobPosting, viewer *models.User) *jobPostingResponse {
	r := &jobPostingResponse{
		ID:          j.PublicID,
		CreatedAt:   j.CreatedAt,
		Company:     j.Company,
		Title:       j.Title,
		Body:        j.Body,
		ApplyURL:    j.ApplyURL,
		Location:    j.Location,
		Remote:      j.Remote,
		CountryCode: j.CountryCode,
		SalaryMin:   j.SalaryMin,
		SalaryMax:   j.SalaryMax,
		Currency:    firstNonEmpty(j.Currency, "USD"),
		Seniority:   string(j.Seniority),
		Language:    firstNonEmpty(j.Language, "en"),
		IsSponsored: j.IsSponsored,
		IsFilled:    j.IsFilled,
		ViewCount:   j.ViewCount,
	}
	if j.PostedBy != nil {
		r.PostedBy = toUserResponse(j.PostedBy, viewer)
	}
	return r
}

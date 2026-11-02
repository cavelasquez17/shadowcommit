// Package models holds the persistence layer schema for ShadowCommit.
package models

import (
	"time"

	"gorm.io/gorm"
)

// Seniority is the rank a user held right before becoming statistically irrelevant.
type Seniority string

const (
	SeniorityIntern    Seniority = "intern"
	SeniorityJunior    Seniority = "junior"
	SeniorityMid       Seniority = "mid"
	SenioritySenior    Seniority = "senior"
	SeniorityStaff     Seniority = "staff"
	SeniorityPrincipal Seniority = "principal"
	SeniorityLead      Seniority = "lead"
	SeniorityCTO       Seniority = "cto"
	SeniorityShadow    Seniority = "shadow"
	SeniorityFreelance Seniority = "freelance_survivor"
)

// CauseOfDeath enumerates the ways a career can end in the current year of our lord.
type CauseOfDeath string

const (
	CauseCopilot        CauseOfDeath = "copilot_did_it_faster"
	CauseCEODemo        CauseOfDeath = "ceo_saw_a_demo"
	CauseShadowExposed  CauseOfDeath = "shadow_work_exposed"
	CausePromptEngineer CauseOfDeath = "replaced_by_a_prompt"
	CauseOffshore       CauseOfDeath = "replaced_by_an_agent_swarm"
	CauseRestructure    CauseOfDeath = "synergistic_restructuring"
	CauseStillAlive     CauseOfDeath = "still_alive_somehow"
	CauseVoluntary      CauseOfDeath = "left_before_they_could_fire_me"
)

// User is a survivor, a casualty, or a person in denial.
type User struct {
	ID        uint           `gorm:"primarykey" json:"-"`
	PublicID  string         `gorm:"size:24;uniqueIndex;not null" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Identity
	GitHubID  *int64 `gorm:"uniqueIndex" json:"-"`
	Username  string `gorm:"size:39;uniqueIndex;not null" json:"username"`
	Email     string `gorm:"size:320;index" json:"-"`
	Password  string `gorm:"size:120" json:"-"`
	AvatarURL string `gorm:"size:512" json:"avatar_url"`

	// The good stuff
	DisplayName  string       `gorm:"size:80" json:"display_name"`
	Bio          string       `gorm:"size:500" json:"bio"`
	CountryCode  string       `gorm:"size:2;index" json:"country_code"`
	Timezone     string       `gorm:"size:64" json:"timezone"`
	Seniority    Seniority    `gorm:"size:24;index" json:"seniority"`
	YearsExp     int          `json:"years_exp"`
	CauseOfDeath CauseOfDeath `gorm:"size:48;index" json:"cause_of_death"`
	Epitaph      string       `gorm:"size:180" json:"epitaph"`
	CurrentRole  string       `gorm:"size:120" json:"current_role"`
	// IsEmployed deliberately carries no `default:` tag. GORM omits zero-valued fields from
	// INSERTs when a default exists, so `default:true` would silently turn every
	// IsEmployed=false user into an employed one. Callers set this explicitly.
	IsEmployed bool `gorm:"not null" json:"is_employed"`
	// FiredAt is when the axe landed. Nil means "not yet, keep refreshing".
	FiredAt *time.Time `json:"fired_at"`

	// Links
	GitHubURL   string `gorm:"size:255" json:"github_url"`
	WebsiteURL  string `gorm:"size:255" json:"website_url"`
	LinkedInURL string `gorm:"size:255" json:"linkedin_url"`

	// Preferences
	UILanguage string `gorm:"size:8;default:en" json:"ui_language"`
	// SpokenLanguages is a comma separated list of BCP-47ish tags: "es,en,pt".
	SpokenLanguages string `gorm:"size:120" json:"spoken_languages"`
	CopiumMode      bool   `gorm:"default:false" json:"copium_mode"`

	// Denormalised counters, because JOINs are also being replaced by AI.
	ThreadCount int `gorm:"default:0" json:"thread_count"`
	PostCount   int `gorm:"default:0" json:"post_count"`
	Reputation  int `gorm:"default:0" json:"reputation"`

	Role string `gorm:"size:16;default:member" json:"role"`

	Repos  []Repo  `gorm:"foreignKey:UserID" json:"repos,omitempty"`
	Badges []Badge `gorm:"many2many:user_badges" json:"badges,omitempty"`
}

// Repo is a link to the code that outlived its author's employment.
type Repo struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	PublicID  string    `gorm:"size:24;uniqueIndex;not null" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UserID    uint      `gorm:"index;not null" json:"-"`

	Name        string `gorm:"size:140;not null" json:"name"`
	URL         string `gorm:"size:512;not null" json:"url"`
	Description string `gorm:"size:300" json:"description"`
	Language    string `gorm:"size:40" json:"language"`
	Stars       int    `json:"stars"`
	// WrittenByAI is the confession field.
	WrittenByAI bool `gorm:"default:false" json:"written_by_ai"`
}

// Badge is a participation trophy for the apocalypse.
type Badge struct {
	ID          uint   `gorm:"primarykey" json:"-"`
	Slug        string `gorm:"size:48;uniqueIndex;not null" json:"slug"`
	Label       string `gorm:"size:64;not null" json:"label"`
	Description string `gorm:"size:240" json:"description"`
	Emoji       string `gorm:"size:16" json:"emoji"`
	// Color is a tailwind-ish hex used by the frontend to scream.
	Color string `gorm:"size:16" json:"color"`
}

// Category groups threads by flavour of despair.
type Category struct {
	ID          uint   `gorm:"primarykey" json:"-"`
	Slug        string `gorm:"size:48;uniqueIndex;not null" json:"slug"`
	Name        string `gorm:"size:64;not null" json:"name"`
	Description string `gorm:"size:240" json:"description"`
	Emoji       string `gorm:"size:16" json:"emoji"`
	Color       string `gorm:"size:16" json:"color"`
	SortOrder   int    `gorm:"default:0" json:"sort_order"`
	ThreadCount int    `gorm:"default:0" json:"thread_count"`
}

// Tag is a free-form label. Users will abuse this immediately.
type Tag struct {
	ID         uint   `gorm:"primarykey" json:"-"`
	Slug       string `gorm:"size:48;uniqueIndex;not null" json:"slug"`
	Label      string `gorm:"size:48;not null" json:"label"`
	UsageCount int    `gorm:"default:0" json:"usage_count"`
}

// Thread is one story of obsolescence.
type Thread struct {
	ID        uint           `gorm:"primarykey" json:"-"`
	PublicID  string         `gorm:"size:24;uniqueIndex;not null" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	AuthorID   uint      `gorm:"index;not null" json:"-"`
	Author     *User     `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	CategoryID uint      `gorm:"index;not null" json:"-"`
	Category   *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`

	Title string `gorm:"size:200;not null" json:"title"`
	Slug  string `gorm:"size:220;index" json:"slug"`
	Body  string `gorm:"type:text;not null" json:"body"`
	// Language is the BCP-47 tag the post was written in, so we can filter the tower of babel.
	Language string `gorm:"size:8;index;default:en" json:"language"`

	// Sarcasm is a 0-100 self-declared score. Nobody rates themselves below 80.
	Sarcasm int `gorm:"default:100" json:"sarcasm"`

	IsPinned bool `gorm:"default:false" json:"is_pinned"`
	IsLocked bool `gorm:"default:false" json:"is_locked"`
	// NSFWCorporate flags stories that name names.
	NSFWCorporate bool `gorm:"default:false" json:"nsfw_corporate"`

	ViewCount     int        `gorm:"default:0" json:"view_count"`
	PostCount     int        `gorm:"default:0" json:"post_count"`
	ReactionCount int        `gorm:"default:0" json:"reaction_count"`
	HotScore      float64    `gorm:"index;default:0" json:"hot_score"`
	LastPostAt    *time.Time `gorm:"index" json:"last_post_at"`

	Tags  []Tag  `gorm:"many2many:thread_tags" json:"tags,omitempty"`
	Posts []Post `gorm:"foreignKey:ThreadID" json:"posts,omitempty"`
}

// Post is a reply. Or a eulogy.
type Post struct {
	ID        uint           `gorm:"primarykey" json:"-"`
	PublicID  string         `gorm:"size:24;uniqueIndex;not null" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	ThreadID uint  `gorm:"index;not null" json:"-"`
	AuthorID uint  `gorm:"index;not null" json:"-"`
	Author   *User `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	// ParentID enables one level of nesting. Reddit-style infinity is a support ticket generator.
	ParentID *uint `gorm:"index" json:"-"`

	Body     string `gorm:"type:text;not null" json:"body"`
	Language string `gorm:"size:8;default:en" json:"language"`

	ReactionCount int  `gorm:"default:0" json:"reaction_count"`
	IsAnswer      bool `gorm:"default:false" json:"is_answer"`
	EditedByAI    bool `gorm:"default:false" json:"edited_by_ai"`
}

// ReactionKind is the emotional vocabulary of the developer in decline.
type ReactionKind string

const (
	ReactionF      ReactionKind = "f"      // 💀 press F
	ReactionRelate ReactionKind = "relate" // 🫠 too real
	ReactionLaugh  ReactionKind = "laugh"  // 😂
	ReactionRobot  ReactionKind = "robot"  // 🤖 the machine did this
	ReactionCopium ReactionKind = "copium" // 🫡 it will be fine
	ReactionBrutal ReactionKind = "brutal" // 🔥 no notes
	ReactionLawyer ReactionKind = "lawyer" // ⚖️ you should sue
)

// Reaction is a single emoji-shaped scream. Unique per (user, target).
type Reaction struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	CreatedAt time.Time `json:"created_at"`

	UserID uint         `gorm:"uniqueIndex:idx_reaction_unique;not null" json:"-"`
	Kind   ReactionKind `gorm:"size:16;uniqueIndex:idx_reaction_unique;not null" json:"kind"`
	// TargetType is "thread" or "post".
	TargetType string `gorm:"size:8;uniqueIndex:idx_reaction_unique;not null" json:"target_type"`
	TargetID   uint   `gorm:"uniqueIndex:idx_reaction_unique;not null" json:"-"`
}

// Mention records an @tag so we can notify the tagged party of their impending doom.
type Mention struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	CreatedAt time.Time `json:"created_at"`

	MentionedUserID uint  `gorm:"index;not null" json:"-"`
	ByUserID        uint  `gorm:"index;not null" json:"-"`
	ThreadID        uint  `gorm:"index;not null" json:"-"`
	PostID          *uint `gorm:"index" json:"-"`
}

// Invite is a golden ticket to the funeral.
type Invite struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	CreatedAt time.Time `json:"created_at"`

	Code        string     `gorm:"size:16;uniqueIndex;not null" json:"code"`
	CreatedByID uint       `gorm:"index;not null" json:"-"`
	UsedByID    *uint      `gorm:"index" json:"-"`
	UsedAt      *time.Time `json:"used_at"`
	// Note is the sales pitch: "come, the water is warm".
	Note      string    `gorm:"size:200" json:"note"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Notification is how we make sure the bad news finds you.
type Notification struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	PublicID  string    `gorm:"size:24;uniqueIndex;not null" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID uint   `gorm:"index;not null" json:"-"`
	Kind   string `gorm:"size:32;not null" json:"kind"`
	Title  string `gorm:"size:200" json:"title"`
	Body   string `gorm:"size:400" json:"body"`
	Link   string `gorm:"size:255" json:"link"`
	IsRead bool   `gorm:"default:false;index" json:"is_read"`
}

// Follow is the social graph, such as it is.
type Follow struct {
	ID          uint      `gorm:"primarykey" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	FollowerID  uint      `gorm:"uniqueIndex:idx_follow_unique;not null" json:"-"`
	FollowingID uint      `gorm:"uniqueIndex:idx_follow_unique;not null" json:"-"`
}

// JobPosting is a paid listing on "Hire the Fallen". Companies pay to reach devs the machine
// couldn't replace. IsSponsored pins it to the top — that's the $200 tier.
type JobPosting struct {
	ID        uint           `gorm:"primarykey" json:"-"`
	PublicID  string         `gorm:"size:24;uniqueIndex;not null" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	PostedByID uint  `gorm:"index;not null" json:"-"`
	PostedBy   *User `gorm:"foreignKey:PostedByID" json:"posted_by,omitempty"`

	Company     string `gorm:"size:120;not null" json:"company"`
	Title       string `gorm:"size:200;not null" json:"title"`
	Body        string `gorm:"type:text;not null" json:"body"`
	ApplyURL    string `gorm:"size:512;not null" json:"apply_url"`
	Location    string `gorm:"size:120" json:"location"`
	Remote      bool   `gorm:"not null" json:"remote"`
	CountryCode string `gorm:"size:2;index" json:"country_code"`

	SalaryMin *int   `json:"salary_min"`
	SalaryMax *int   `json:"salary_max"`
	Currency  string `gorm:"size:3;default:USD" json:"currency"`

	Seniority Seniority `gorm:"size:24;index" json:"seniority"`
	Language  string    `gorm:"size:8;index;default:en" json:"language"`

	// IsSponsored is the monetization lever. Sponsored jobs pin to the top of the feed.
	// In MVP this is toggled manually after payment confirmation. Stripe webhook later.
	IsSponsored bool `gorm:"default:false;index" json:"is_sponsored"`
	IsFilled    bool `gorm:"default:false" json:"is_filled"`

	ViewCount int `gorm:"default:0" json:"view_count"`
}

// AllModels is the migration manifest.
func AllModels() []any {
	return []any{
		&User{}, &Repo{}, &Badge{}, &Category{}, &Tag{},
		&Thread{}, &Post{}, &Reaction{}, &Mention{},
		&Invite{}, &Notification{}, &Follow{}, &JobPosting{},
	}
}

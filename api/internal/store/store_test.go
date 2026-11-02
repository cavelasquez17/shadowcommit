package store_test

import (
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
	"github.com/cavelasquez17/shadowcommit/api/internal/testutil"
)

func testDB(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{DBDriver: "sqlite", DSN: testutil.MemoryDSN(t)}
}

// --- GORM footgun guards ---

// TestUserIsEmployedPersistsFalse guards a GORM footgun: a `default:` tag makes GORM omit
// zero-valued fields from INSERTs, so `default:true` on IsEmployed silently resurrected the
// employment of every unemployed member. The whole Wall of the Fallen depends on this being false.
func TestUserIsEmployedPersistsFalse(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	u := models.User{
		PublicID:   store.NewPublicID("usr"),
		Username:   "obsolete",
		IsEmployed: false,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	var got models.User
	if err := db.Where("username = ?", "obsolete").First(&got).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got.IsEmployed {
		t.Fatal("IsEmployed was persisted as true; a `default:` tag is swallowing the false value")
	}
}

// --- NewPublicID ---

func TestNewPublicIDIsPrefixedAndUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool, 500)
	for range 500 {
		id := store.NewPublicID("thr")
		if !strings.HasPrefix(id, "thr_") {
			t.Fatalf("missing prefix: %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id generated: %q", id)
		}
		seen[id] = true
	}
}

func TestNewPublicIDLengthIsConsistent(t *testing.T) {
	t.Parallel()

	// 10 random bytes → 16 chars in base32 (no padding), plus prefix + underscore.
	id := store.NewPublicID("usr")
	if len(id) != 3+1+16 { // "usr" + "_" + 16 chars
		t.Fatalf("unexpected id length: got %d, want 20", len(id))
	}
}

func TestNewPublicIDEmptyPrefixStillInjectsUnderscore(t *testing.T) {
	t.Parallel()

	// NewPublicID always does prefix + "_" + encoding, so an empty prefix yields
	// "_<encoded>". This is a known cosmetic quirk, not a bug — no caller passes "".
	id := store.NewPublicID("")
	if !strings.HasPrefix(id, "_") {
		t.Fatalf("empty prefix should still inject underscore separator: %q", id)
	}
}

// --- Open / driver selection ---

func TestOpenRejectsUnknownDriver(t *testing.T) {
	t.Parallel()

	if _, err := store.Open(&config.Config{DBDriver: "oracle", DSN: "x"}); err == nil {
		t.Fatal("expected an error for an unsupported driver")
	}
}

func TestOpenMigratesAllModels(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Every model in AllModels() should have a table after Open.
	for _, m := range models.AllModels() {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			t.Fatalf("parse model %T: %v", m, err)
		}
		table := stmt.Schema.Table
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil {
			t.Fatalf("table %s not migrated: %v", table, err)
		}
	}
}

// --- Thread lifecycle ---

func TestThreadCreateWithCategoryAndAuthor(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	cat := models.Category{Slug: "the-incident", Name: "The Incident"}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	author := models.User{PublicID: store.NewPublicID("usr"), Username: "ghost1"}
	if err := db.Create(&author).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	thread := models.Thread{
		PublicID:   store.NewPublicID("thr"),
		AuthorID:   author.ID,
		CategoryID: cat.ID,
		Title:      "Copilot wrote my PR and my boss noticed",
		Body:       "It was a one-liner. The AI added a semicolon to Python.",
		Slug:       "copilot-wrote-my-pr",
	}
	if err := db.Create(&thread).Error; err != nil {
		t.Fatalf("create thread: %v", err)
	}

	var got models.Thread
	if err := db.Preload("Author").Preload("Category").Where("public_id = ?", thread.PublicID).First(&got).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got.Title != thread.Title {
		t.Fatalf("title mismatch: got %q", got.Title)
	}
	if got.Author.Username != "ghost1" {
		t.Fatalf("author not preloaded: got %q", got.Author.Username)
	}
	if got.Category.Slug != "the-incident" {
		t.Fatalf("category not preloaded: got %q", got.Category.Slug)
	}
}

func TestThreadSoftDeleteHidesFromDefaultQueries(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	cat := models.Category{Slug: "off-topic", Name: "Off Topic"}
	db.Create(&cat)
	author := models.User{PublicID: store.NewPublicID("usr"), Username: "ghost2"}
	db.Create(&author)

	thread := models.Thread{
		PublicID: store.NewPublicID("thr"), AuthorID: author.ID, CategoryID: cat.ID,
		Title: "Deleted thread", Body: "should not appear",
	}
	db.Create(&thread)

	if err := db.Delete(&thread).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	// Default query excludes soft-deleted rows.
	var count int64
	db.Model(&models.Thread{}).Where("public_id = ?", thread.PublicID).Count(&count)
	if count != 0 {
		t.Fatalf("soft-deleted thread should not appear in default query, got count=%d", count)
	}

	// Unscoped reveals it.
	var raw models.Thread
	if err := db.Unscoped().Where("public_id = ?", thread.PublicID).First(&raw).Error; err != nil {
		t.Fatalf("soft-deleted thread should be recoverable via Unscoped: %v", err)
	}
	if raw.DeletedAt.Time.IsZero() {
		t.Fatal("DeletedAt should be set after soft delete")
	}
}

// --- Post lifecycle ---

func TestPostBelongsToThread(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	cat := models.Category{Slug: "ai-did-this", Name: "AI Did This"}
	db.Create(&cat)
	author := models.User{PublicID: store.NewPublicID("usr"), Username: "ghost3"}
	db.Create(&author)
	thread := models.Thread{
		PublicID: store.NewPublicID("thr"), AuthorID: author.ID, CategoryID: cat.ID,
		Title: "Thread", Body: "Body",
	}
	db.Create(&thread)

	post := models.Post{
		PublicID: store.NewPublicID("pst"), ThreadID: thread.ID, AuthorID: author.ID,
		Body: "First reply, written by a human (allegedly)",
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	var got models.Post
	if err := db.Preload("Author").Where("public_id = ?", post.PublicID).First(&got).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got.ThreadID != thread.ID {
		t.Fatalf("post thread_id mismatch: got %d, want %d", got.ThreadID, thread.ID)
	}
}

func TestPostParentIDEnablesOneLevelNesting(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	cat := models.Category{Slug: "shadow-work", Name: "Shadow Work"}
	db.Create(&cat)
	author := models.User{PublicID: store.NewPublicID("usr"), Username: "ghost4"}
	db.Create(&author)
	thread := models.Thread{
		PublicID: store.NewPublicID("thr"), AuthorID: author.ID, CategoryID: cat.ID,
		Title: "Nested", Body: "Body",
	}
	db.Create(&thread)

	parent := models.Post{PublicID: store.NewPublicID("pst"), ThreadID: thread.ID, AuthorID: author.ID, Body: "Parent"}
	db.Create(&parent)

	child := models.Post{PublicID: store.NewPublicID("pst"), ThreadID: thread.ID, AuthorID: author.ID, Body: "Child", ParentID: &parent.ID}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child post: %v", err)
	}

	var got models.Post
	db.Where("public_id = ?", child.PublicID).First(&got)
	if got.ParentID == nil || *got.ParentID != parent.ID {
		t.Fatalf("parent_id not set correctly: got %v", got.ParentID)
	}
}

// --- Reaction uniqueness ---

func TestReactionUniquePerUserTargetKind(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	user := models.User{PublicID: store.NewPublicID("usr"), Username: "reactor"}
	db.Create(&user)
	cat := models.Category{Slug: "the-incident", Name: "The Incident"}
	db.Create(&cat)
	thread := models.Thread{
		PublicID: store.NewPublicID("thr"), AuthorID: user.ID, CategoryID: cat.ID,
		Title: "React test", Body: "Body",
	}
	db.Create(&thread)

	r1 := models.Reaction{UserID: user.ID, Kind: models.ReactionF, TargetType: "thread", TargetID: thread.ID}
	if err := db.Create(&r1).Error; err != nil {
		t.Fatalf("create reaction: %v", err)
	}

	// Same user + same target + same kind must fail.
	r2 := models.Reaction{UserID: user.ID, Kind: models.ReactionF, TargetType: "thread", TargetID: thread.ID}
	if err := db.Create(&r2).Error; err == nil {
		t.Fatal("expected unique constraint violation for duplicate reaction")
	}

	// Same user + same target + different kind is allowed.
	r3 := models.Reaction{UserID: user.ID, Kind: models.ReactionLaugh, TargetType: "thread", TargetID: thread.ID}
	if err := db.Create(&r3).Error; err != nil {
		t.Fatalf("different kind should be allowed: %v", err)
	}
}

// --- Follow uniqueness ---

func TestFollowUniquePerPair(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	a := models.User{PublicID: store.NewPublicID("usr"), Username: "follower"}
	b := models.User{PublicID: store.NewPublicID("usr"), Username: "followed"}
	db.Create(&a)
	db.Create(&b)

	f1 := models.Follow{FollowerID: a.ID, FollowingID: b.ID}
	if err := db.Create(&f1).Error; err != nil {
		t.Fatalf("create follow: %v", err)
	}

	f2 := models.Follow{FollowerID: a.ID, FollowingID: b.ID}
	if err := db.Create(&f2).Error; err == nil {
		t.Fatal("expected unique constraint violation for duplicate follow")
	}
}

// --- Invite lifecycle ---

func TestInviteCodeIsUniqueAndExpirable(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	creator := models.User{PublicID: store.NewPublicID("usr"), Username: "inviter"}
	db.Create(&creator)

	invite := models.Invite{
		Code:        "SC-TEST-CODE-01",
		CreatedByID: creator.ID,
		ExpiresAt:   time.Now().Add(30 * 24 * time.Hour),
	}
	if err := db.Create(&invite).Error; err != nil {
		t.Fatalf("create invite: %v", err)
	}

	// Duplicate code must fail.
	dup := models.Invite{Code: "SC-TEST-CODE-01", CreatedByID: creator.ID, ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(&dup).Error; err == nil {
		t.Fatal("expected unique constraint violation for duplicate invite code")
	}

	// Marking as used sets UsedByID and UsedAt.
	now := time.Now()
	invite.UsedByID = &creator.ID
	invite.UsedAt = &now
	if err := db.Save(&invite).Error; err != nil {
		t.Fatalf("mark used: %v", err)
	}

	var got models.Invite
	db.Where("code = ?", "SC-TEST-CODE-01").First(&got)
	if got.UsedByID == nil {
		t.Fatal("UsedByID should be set after marking used")
	}
}

// --- Tag many2many ---

func TestThreadTagsManyToMany(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	cat := models.Category{Slug: "off-topic", Name: "Off Topic"}
	db.Create(&cat)
	author := models.User{PublicID: store.NewPublicID("usr"), Username: "tagger"}
	db.Create(&author)

	tag1 := models.Tag{Slug: "copilot", Label: "Copilot"}
	tag2 := models.Tag{Slug: "python", Label: "Python"}
	db.Create(&tag1)
	db.Create(&tag2)

	thread := models.Thread{
		PublicID: store.NewPublicID("thr"), AuthorID: author.ID, CategoryID: cat.ID,
		Title: "Tagged thread", Body: "Body", Tags: []models.Tag{tag1, tag2},
	}
	if err := db.Create(&thread).Error; err != nil {
		t.Fatalf("create thread with tags: %v", err)
	}

	var got models.Thread
	db.Preload("Tags").Where("public_id = ?", thread.PublicID).First(&got)
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(got.Tags))
	}
}

// --- Badge many2many ---

func TestUserBadgesManyToMany(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	badge := models.Badge{Slug: "certified-obsolete", Label: "Certified Obsolete", Emoji: "📜"}
	db.Create(&badge)

	user := models.User{
		PublicID: store.NewPublicID("usr"), Username: "badged",
		Badges: []models.Badge{badge},
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user with badge: %v", err)
	}

	var got models.User
	db.Preload("Badges").Where("username = ?", "badged").First(&got)
	if len(got.Badges) != 1 || got.Badges[0].Slug != "certified-obsolete" {
		t.Fatalf("badge not associated: %+v", got.Badges)
	}
}

// --- User soft delete ---

func TestUserSoftDeleteHidesFromDirectory(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	user := models.User{PublicID: store.NewPublicID("usr"), Username: "ghost-deleted"}
	db.Create(&user)

	if err := db.Delete(&user).Error; err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	var count int64
	db.Model(&models.User{}).Where("username = ?", "ghost-deleted").Count(&count)
	if count != 0 {
		t.Fatalf("soft-deleted user should not appear in default query, got count=%d", count)
	}
}

// --- Repo ownership ---

func TestRepoBelongsToUser(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	user := models.User{PublicID: store.NewPublicID("usr"), Username: "repo-owner"}
	db.Create(&user)

	repo := models.Repo{
		PublicID: store.NewPublicID("repo"), UserID: user.ID,
		Name: "my-side-project", URL: "https://github.com/user/repo",
		Language: "Go", Stars: 42, WrittenByAI: true,
	}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatalf("create repo: %v", err)
	}

	var got models.Repo
	db.Where("public_id = ?", repo.PublicID).First(&got)
	if got.UserID != user.ID {
		t.Fatalf("repo user_id mismatch: got %d, want %d", got.UserID, user.ID)
	}
	if !got.WrittenByAI {
		t.Fatal("WrittenByAI should be true — the AI wrote this repo and we should know")
	}
}

// --- Notification ---

func TestNotificationPersistsUnread(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	user := models.User{PublicID: store.NewPublicID("usr"), Username: "notified"}
	db.Create(&user)

	notif := models.Notification{
		PublicID: store.NewPublicID("ntf"), UserID: user.ID,
		Kind: "mention", Title: "You were mentioned", Body: "@notified check this out",
		Link: "/t/some-thread",
	}
	if err := db.Create(&notif).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}

	var got models.Notification
	db.Where("public_id = ?", notif.PublicID).First(&got)
	if got.IsRead {
		t.Fatal("notification should default to unread (IsRead=false)")
	}
}

// --- Mention ---

func TestMentionLinksUsersAndThread(t *testing.T) {
	t.Parallel()

	db, err := store.Open(testDB(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	cat := models.Category{Slug: "the-incident", Name: "The Incident"}
	db.Create(&cat)
	author := models.User{PublicID: store.NewPublicID("usr"), Username: "mentioner"}
	target := models.User{PublicID: store.NewPublicID("usr"), Username: "mentioned"}
	db.Create(&author)
	db.Create(&target)
	thread := models.Thread{
		PublicID: store.NewPublicID("thr"), AuthorID: author.ID, CategoryID: cat.ID,
		Title: "Mention thread", Body: "Body",
	}
	db.Create(&thread)

	mention := models.Mention{
		MentionedUserID: target.ID, ByUserID: author.ID, ThreadID: thread.ID,
	}
	if err := db.Create(&mention).Error; err != nil {
		t.Fatalf("create mention: %v", err)
	}

	var got models.Mention
	db.Where("mentioned_user_id = ?", target.ID).First(&got)
	if got.ByUserID != author.ID {
		t.Fatalf("mention by_user_id mismatch: got %d, want %d", got.ByUserID, author.ID)
	}
}

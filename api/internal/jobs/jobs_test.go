package jobs

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
	"github.com/cavelasquez17/shadowcommit/api/internal/testutil"
)

var now = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := store.Open(&config.Config{DBDriver: "sqlite", DSN: testutil.MemoryDSN(t)})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func mkUser(t *testing.T, db *gorm.DB, name string) models.User {
	t.Helper()
	u := models.User{PublicID: store.NewPublicID("usr"), Username: name}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func mkThread(t *testing.T, db *gorm.DB, author models.User, cat models.Category, title string, reactions int, created time.Time) models.Thread {
	t.Helper()
	th := models.Thread{
		PublicID: store.NewPublicID("thr"), AuthorID: author.ID, CategoryID: cat.ID,
		Title: title, Slug: title, Body: "body long enough to be valid", ReactionCount: reactions,
	}
	if err := db.Create(&th).Error; err != nil {
		t.Fatal(err)
	}
	// CreatedAt is set by GORM on insert; force the age we want.
	if err := db.Model(&th).UpdateColumn("created_at", created).Error; err != nil {
		t.Fatal(err)
	}
	return th
}

func mkCategory(t *testing.T, db *gorm.DB) models.Category {
	t.Helper()
	c := models.Category{Slug: "the-incident", Name: "The Incident"}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRefreshHotScoresDecaysOldThreads(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	author, cat := mkUser(t, db, "author"), mkCategory(t, db)
	old := mkThread(t, db, author, cat, "old-but-popular", 100, now.AddDate(0, 0, -30))
	fresh := mkThread(t, db, author, cat, "fresh-and-modest", 5, now.Add(-2*time.Hour))
	// Simulate the stale state the job exists to fix: both carry an inflated score.
	db.Model(&models.Thread{}).Where("1=1").UpdateColumn("hot_score", 99.0)

	n, err := RefreshHotScores(context.Background(), db, now)
	if err != nil || n != 2 {
		t.Fatalf("refresh = (%d, %v), want (2, nil)", n, err)
	}
	db.First(&old, old.ID)
	db.First(&fresh, fresh.ID)
	if old.HotScore >= 99 || fresh.HotScore >= 99 {
		t.Errorf("scores were not recomputed: old=%v fresh=%v", old.HotScore, fresh.HotScore)
	}
	if old.HotScore >= fresh.HotScore {
		t.Errorf("a 30-day-old thread should rank below a fresh one: old=%v fresh=%v", old.HotScore, fresh.HotScore)
	}
}

func TestRefreshHotScoresIgnoresDeletedThreads(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	author, cat := mkUser(t, db, "author"), mkCategory(t, db)
	th := mkThread(t, db, author, cat, "gone", 1, now)
	db.Delete(&th)
	if n, err := RefreshHotScores(context.Background(), db, now); err != nil || n != 0 {
		t.Fatalf("refresh = (%d, %v), want (0, nil)", n, err)
	}
}

func TestSendWeeklyDigest(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	a, b := mkUser(t, db, "alice"), mkUser(t, db, "bob")
	gone := mkUser(t, db, "gone")
	db.Delete(&gone)
	cat := mkCategory(t, db)
	mkThread(t, db, a, cat, "minor", 2, now.AddDate(0, 0, -2))
	best := mkThread(t, db, a, cat, "the-big-one", 40, now.AddDate(0, 0, -3))
	mkThread(t, db, a, cat, "too-old-to-count", 999, now.AddDate(0, 0, -20))

	n, err := SendWeeklyDigest(context.Background(), db, now)
	if err != nil || n != 2 {
		t.Fatalf("digest = (%d, %v), want 2 notifications (deleted user excluded)", n, err)
	}
	var got []models.Notification
	db.Where("kind = ?", DigestKind).Find(&got)
	for _, nt := range got {
		if nt.Body != "the-big-one" || nt.Link != "/t/"+best.PublicID {
			t.Errorf("digest points at the wrong thread: %+v", nt)
		}
		if nt.UserID != a.ID && nt.UserID != b.ID {
			t.Errorf("digest sent to an unexpected user %d", nt.UserID)
		}
	}

	if n, _ := SendWeeklyDigest(context.Background(), db, now.Add(time.Hour)); n != 0 {
		t.Errorf("running twice must not spam: second run created %d", n)
	}
	if n, _ := SendWeeklyDigest(context.Background(), db, now.AddDate(0, 0, 7)); n != 0 {
		// No thread in the new 7-day window, so still nothing to announce.
		t.Errorf("a quiet week should send nothing, got %d", n)
	}
}

func TestSendWeeklyDigestQuietWeek(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	mkUser(t, db, "alice")
	if n, err := SendWeeklyDigest(context.Background(), db, now); err != nil || n != 0 {
		t.Fatalf("digest with no threads = (%d, %v), want (0, nil)", n, err)
	}
}

package seed

import (
	"testing"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
	"github.com/cavelasquez17/shadowcommit/api/internal/testutil"
)

// Accounts created before avatars became local still point at the third-party service. The
// migration rewrites only those, and running it twice changes nothing.
func TestMigrateAvatarsRewritesOnlyThirdPartyURLs(t *testing.T) {
	t.Parallel()
	db, err := store.Open(&config.Config{DBDriver: "sqlite", DSN: testutil.MemoryDSN(t)})
	if err != nil {
		t.Fatal(err)
	}
	users := []models.User{
		{PublicID: store.NewPublicID("usr"), Username: "old_one", AvatarURL: "https://api.dicebear.com/9.x/bottts-neutral/svg?seed=old_one"},
		{PublicID: store.NewPublicID("usr"), Username: "real_gh", AvatarURL: "https://avatars.githubusercontent.com/u/42"},
		{PublicID: store.NewPublicID("usr"), Username: "no_avatar", AvatarURL: ""},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateAvatars(db); err != nil {
		t.Fatal(err)
	}
	if got := avatarOf(t, db, "old_one"); got != "/api/avatars/old_one" {
		t.Errorf("third-party avatar became %q, want the local path", got)
	}
	if got := avatarOf(t, db, "real_gh"); got != "https://avatars.githubusercontent.com/u/42" {
		t.Errorf("a real GitHub avatar was changed to %q", got)
	}
	if got := avatarOf(t, db, "no_avatar"); got != "" {
		t.Errorf("an empty avatar was filled in with %q", got)
	}

	if err := MigrateAvatars(db); err != nil {
		t.Fatal(err)
	}
	if got := avatarOf(t, db, "old_one"); got != "/api/avatars/old_one" {
		t.Errorf("second run changed it to %q", got)
	}
}

func avatarOf(t *testing.T, db *gorm.DB, name string) string {
	t.Helper()
	var u models.User
	if err := db.Where("username = ?", name).First(&u).Error; err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return u.AvatarURL
}

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/moodphoto"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
	"github.com/cavelasquez17/shadowcommit/api/internal/testutil"
)

// A backup must be a complete, openable database that contains what was in the live one, and it
// must be taken without changing the live database.
func TestBackupIsARestorableCopy(t *testing.T) {
	t.Parallel()
	db, err := store.Open(&config.Config{DBDriver: "sqlite", DSN: testutil.MemoryDSN(t)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.User{PublicID: store.NewPublicID("usr"), Username: "keeper"}).Error; err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "backup.db")
	if err := backupDatabase(db, dest); err != nil {
		t.Fatalf("backup: %v", err)
	}

	restored, err := store.Open(&config.Config{DBDriver: "sqlite", DSN: dest})
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	var n int64
	restored.Model(&models.User{}).Where("username = ?", "keeper").Count(&n)
	if n != 1 {
		t.Errorf("backup has %d copies of the user, want 1", n)
	}

	// The live database is untouched and still works.
	var live int64
	db.Model(&models.User{}).Count(&live)
	if live != 1 {
		t.Errorf("live database now has %d users, want 1", live)
	}
}

func TestBackupRefusesToOverwriteAnExistingFile(t *testing.T) {
	t.Parallel()
	db, err := store.Open(&config.Config{DBDriver: "sqlite", DSN: testutil.MemoryDSN(t)})
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "existing.db")
	if err := backupDatabase(db, dest); err != nil {
		t.Fatal(err)
	}
	if err := backupDatabase(db, dest); err == nil {
		t.Error("a second backup to the same name must fail, not silently replace the first")
	}
}

// A file that is not a database, or a damaged one, must fail the check instead of being reported
// as a good backup.
func TestVerifyBackupRejectsADamagedFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "damaged.db")
	if err := os.WriteFile(path, []byte("this is not sqlite at all, not even a little"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyBackup(path); err == nil {
		t.Error("a damaged file must not pass as a verified backup")
	}
}

func TestBackupPairContainsDatabaseAndMoodPhotos(t *testing.T) {
	t.Parallel()
	db, err := store.Open(&config.Config{DBDriver: "sqlite", DSN: testutil.MemoryDSN(t)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.User{PublicID: store.NewPublicID("usr"), Username: "mood-backup"}).Error; err != nil {
		t.Fatal(err)
	}
	photos := moodphoto.New(t.TempDir())
	var key string
	if err := photos.WithExclusive(func() error {
		var err error
		key, _, err = photos.Save(bytes.NewReader(backupPNG(t)))
		return err
	}); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "snapshot.db")
	photoPath := filepath.Join(dir, "snapshot.photos.tar.gz")
	if err := backupPair(db, dbPath, photoPath, photos); err != nil {
		t.Fatalf("backup pair: %v", err)
	}
	if err := verifyBackup(dbPath); err != nil {
		t.Fatalf("database backup: %v", err)
	}

	archive, err := os.Open(photoPath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	gz, err := gzip.NewReader(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	header, err := tar.NewReader(gz).Next()
	if err != nil {
		t.Fatalf("photo archive entry: %v", err)
	}
	if header.Name != key {
		t.Errorf("photo archive contains %q, want %q", header.Name, key)
	}
}

func TestBackupPairRemovesDatabaseCopyWhenPhotoArchiveFails(t *testing.T) {
	t.Parallel()
	db, err := store.Open(&config.Config{DBDriver: "sqlite", DSN: testutil.MemoryDSN(t)})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "snapshot.db")
	photoPath := filepath.Join(dir, "existing.photos.tar.gz")
	if err := os.WriteFile(photoPath, []byte("preserve existing archive"), 0600); err != nil {
		t.Fatal(err)
	}
	err = backupPair(db, dbPath, photoPath, moodphoto.New(t.TempDir()))
	if err == nil {
		t.Fatal("expected refusing to replace an existing photo archive")
	}
	if _, statErr := os.Stat(dbPath); !os.IsNotExist(statErr) {
		t.Errorf("incomplete database copy remains: %v", statErr)
	}
	if got, readErr := os.ReadFile(photoPath); readErr != nil || string(got) != "preserve existing archive" {
		t.Errorf("existing archive changed: %q, %v", got, readErr)
	}
}

func backupPNG(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

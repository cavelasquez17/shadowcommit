package main

import (
	"errors"
	"fmt"
	"os"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/moodphoto"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

// backupDatabase writes a complete copy of the live SQLite database to dest with VACUUM INTO, which
// reads consistently while the app keeps serving. It refuses an existing file, so a backup can never
// silently replace an older one. The copy is opened and checked before the call succeeds.
func backupDatabase(db *gorm.DB, dest string) error {
	if err := db.Exec("VACUUM INTO ?", dest).Error; err != nil {
		return fmt.Errorf("writing backup to %s: %w", dest, err)
	}
	if err := verifyBackup(dest); err != nil {
		if removeErr := os.Remove(dest); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("remove invalid database backup: %w", removeErr))
		}
		return err
	}
	return nil
}

// backupPair snapshots the database and its photo files under the same cross-process store lock.
// If the photo archive cannot be completed, it removes the new database copy so no mismatched pair
// is left behind.
func backupPair(db *gorm.DB, dbDest, photoDest string, photos *moodphoto.Store) error {
	return photos.WithExclusive(func() error {
		if err := backupDatabase(db, dbDest); err != nil {
			return err
		}
		if err := photos.BackupLocked(photoDest); err != nil {
			if removeErr := os.Remove(dbDest); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				return errors.Join(err, fmt.Errorf("remove incomplete database backup: %w", removeErr))
			}
			return err
		}
		return nil
	})
}

func verifyBackup(path string) error {
	copied, err := store.Open(&config.Config{DBDriver: "sqlite", DSN: path})
	if err != nil {
		return fmt.Errorf("opening backup %s: %w", path, err)
	}
	var verdict string
	if err := copied.Raw("PRAGMA quick_check").Scan(&verdict).Error; err != nil {
		return fmt.Errorf("checking backup %s: %w", path, err)
	}
	if verdict != "ok" {
		return fmt.Errorf("backup %s failed its integrity check: %s", path, verdict)
	}
	return nil
}

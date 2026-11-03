package main

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

// backupDatabase writes a complete copy of the live SQLite database to dest with VACUUM INTO, which
// reads consistently while the app keeps serving. It refuses an existing file, so a backup can never
// silently replace an older one. The copy is opened and checked before the call succeeds.
func backupDatabase(db *gorm.DB, dest string) error {
	if err := db.Exec("VACUUM INTO ?", dest).Error; err != nil {
		return fmt.Errorf("writing backup to %s: %w", dest, err)
	}
	return verifyBackup(dest)
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

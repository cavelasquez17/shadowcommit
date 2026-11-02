// Package store owns the database connection and schema migration.
package store

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

// Open connects to the configured database and runs AutoMigrate.
func Open(cfg *config.Config) (*gorm.DB, error) {
	// "record not found" is a control-flow signal all over this codebase, not an incident.
	// Logging it drowns out the errors that actually matter.
	gcfg := &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "[sql] ", log.LstdFlags), logger.Config{
			SlowThreshold:             300 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		}),
		SkipDefaultTransaction: true,
	}

	var db *gorm.DB
	var err error
	switch cfg.DBDriver {
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DSN), gcfg)
	case "sqlite":
		dsn := cfg.DSN
		if !strings.Contains(dsn, "?") {
			dsn += "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
		}
		db, err = gorm.Open(sqlite.Open(dsn), gcfg)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q (want sqlite or postgres)", cfg.DBDriver)
	}
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

var idEncoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding)

// NewPublicID returns an opaque, URL-safe, non-sequential identifier.
// Exposing auto-increment IDs is how competitors count your users.
func NewPublicID(prefix string) string {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed, the machines have won: %v", err))
	}
	return prefix + "_" + idEncoding.EncodeToString(b)
}

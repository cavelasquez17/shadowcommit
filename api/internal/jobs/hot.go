package jobs

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/ranking"
)

// RefreshHotScores recomputes every live thread's hot_score against now. The score is
// otherwise only written on create and on reaction, so without this an old thread keeps
// its peak score forever and never leaves the top of the "hot" feed. It returns the
// number of threads updated.
func RefreshHotScores(ctx context.Context, db *gorm.DB, now time.Time) (int, error) {
	var rows []struct {
		ID            uint
		ReactionCount int
		CreatedAt     time.Time
	}
	if err := db.WithContext(ctx).Model(&models.Thread{}).
		Select("id, reaction_count, created_at").Find(&rows).Error; err != nil {
		return 0, err
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, r := range rows {
			score := ranking.HotScore(r.ReactionCount, r.CreatedAt, now)
			if err := tx.Model(&models.Thread{}).Where("id = ?", r.ID).
				UpdateColumn("hot_score", score).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return len(rows), err
}

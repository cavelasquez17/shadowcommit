package jobs

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

// DigestKind is the notification kind used by the weekly digest.
const DigestKind = "weekly_digest"

// SendWeeklyDigest tells every member about the most-reacted thread of the past 7 days.
// It is safe to run twice: members who already received a digest in the last 6 days are
// skipped. Returns how many notifications were created.
func SendWeeklyDigest(ctx context.Context, db *gorm.DB, now time.Time) (int, error) {
	db = db.WithContext(ctx)

	var top models.Thread
	err := db.Where("created_at > ?", now.AddDate(0, 0, -7)).
		Order("reaction_count DESC, post_count DESC, created_at DESC").First(&top).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil // a quiet week: nothing worth announcing
	}
	if err != nil {
		return 0, err
	}

	var userIDs []uint
	if err := db.Model(&models.User{}).
		Where(`id NOT IN (SELECT user_id FROM notifications WHERE kind = ? AND created_at > ?)`,
			DigestKind, now.AddDate(0, 0, -6)).
		Pluck("id", &userIDs).Error; err != nil {
		return 0, err
	}
	if len(userIDs) == 0 {
		return 0, nil
	}

	batch := make([]models.Notification, 0, len(userIDs))
	for _, id := range userIDs {
		batch = append(batch, models.Notification{
			PublicID: store.NewPublicID("ntf"), UserID: id, Kind: DigestKind,
			Title: "This week's top casualty",
			Body:  top.Title,
			Link:  "/t/" + top.PublicID,
		})
	}
	if err := db.CreateInBatches(batch, 200).Error; err != nil {
		return 0, err
	}
	return len(batch), nil
}

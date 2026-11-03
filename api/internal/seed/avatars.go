package seed

import (
	"strings"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/avatar"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

const thirdPartyAvatarPrefix = "https://api.dicebear.com/"

// MigrateAvatars points accounts created before default avatars were local at the local endpoint.
// It touches only rows that still use the third-party service, so it is safe to run on every boot.
func MigrateAvatars(db *gorm.DB) error {
	var users []models.User
	if err := db.Where("avatar_url LIKE ?", thirdPartyAvatarPrefix+"%").Find(&users).Error; err != nil {
		return err
	}
	for i := range users {
		if !strings.HasPrefix(users[i].AvatarURL, thirdPartyAvatarPrefix) {
			continue
		}
		if err := db.Model(&users[i]).UpdateColumn("avatar_url", avatar.URLPath(users[i].Username)).Error; err != nil {
			return err
		}
	}
	return nil
}

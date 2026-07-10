package store

import (
	"context"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// SeedDefaults مقادیر پیش‌فرض تنظیمات را تنها در صورت نبودِ کلید ست می‌کند.
// (یک بار هنگام استارت اجرا می‌شود؛ مقادیر موجود را بازنویسی نمی‌کند.)
func (s *Store) SeedDefaults(ctx context.Context) {
	defaults := map[string]string{
		models.SettingBotActive:          "true",
		models.SettingShowSearch:         "true",
		models.SettingShowResendButton:   "true",
		models.SettingFreeDownloads:      "0",
		models.SettingAutoDeleteDefault:  "0",
		models.SettingSpamDelay:          "2",
		models.SettingBroadcastInterval:  "0",
		models.SettingForwardLockDefault: "false",
		models.SettingAntiFilterDefault:  "false",
		models.SettingUserUpload:         "false",
		models.SettingAutoApproveFiles:   "false",
		models.SettingSubRequired:        "false",
	}
	existing := s.GetAllSettings(ctx)
	for k, v := range defaults {
		if _, ok := existing[k]; !ok {
			s.logErr("SeedDefaults", s.SetSetting(ctx, k, v))
		}
	}
	// پیشوند اختصاصی ربات (مثل ux، kj) — یک‌بار خودکار ساخته می‌شود.
	s.EnsureCodePrefix(ctx)
}

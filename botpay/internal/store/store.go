// Package store مدل‌های DB و repository لایه botpay.
//
// متدهای repository بر اساس دامنه در چند فایل تقسیم شده‌اند:
//   - wallet_repo.go    عملیات کیف پول و تراکنش‌ها
//   - invoice_repo.go   عملیات فاکتور واریز
//   - withdraw_repo.go  عملیات برداشت
//   - transfer.go       انتقال داخلی و آمار
package store

import (
	"context"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// Store دسترسی به پایگاه‌داده‌ی botpay را کپسوله می‌کند.
type Store struct{ db *gorm.DB }

// New یک Store جدید روی اتصال gorm می‌سازد.
func New(db *gorm.DB) *Store { return &Store{db: db} }

// DB دسترسی خام به gorm (برای queryهای cross-table مثل validation سرویس).
func (s *Store) DB() *gorm.DB { return s.db }

// ValidateServiceID بررسی می‌کند که یک service_id معتبر است.
// سرویس‌های اصلی پلتفرم همیشه معتبرند.
// سرویس‌های bot instance با فرمت "bot_<BotID>" باید یک instance فعال در DB داشته باشند.
func (s *Store) ValidateServiceID(ctx context.Context, serviceID string) bool {
	if serviceID == "" {
		return false
	}
	switch serviceID {
	case "botmanager", "apimanager", "botpay", "ads-bot",
		"community-service", "fraud-engine", "revenue-service":
		return true
	}
	if !strings.HasPrefix(serviceID, "bot_") {
		return false
	}
	botID, err := strconv.ParseInt(strings.TrimPrefix(serviceID, "bot_"), 10, 64)
	if err != nil {
		return false
	}
	var count int64
	s.db.WithContext(ctx).
		Table("bot_instances").
		Where("bot_id = ? AND status <> ?", botID, "deleted").
		Count(&count)
	return count > 0
}

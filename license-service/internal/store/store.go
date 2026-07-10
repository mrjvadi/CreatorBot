package store

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Store دسترسی DB سرویس لایسنس.
type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *Store { return &Store{db: db} }

// AutoMigrate schema این سرویس را می‌سازد/به‌روز می‌کند.
// (مثل بقیه‌ی سرویس‌های مرکزی، این هم AutoMigrate مستقل خودش را دارد —
// رجوع کنید به گزارش امنیتی برای ریسک شناخته‌شده‌ی «migration drift».)
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&License{})
}

func (s *Store) Create(ctx context.Context, l *License) error {
	return s.db.WithContext(ctx).Create(l).Error
}

func (s *Store) FindByBotID(ctx context.Context, botID int64) (*License, error) {
	var l License
	err := s.db.WithContext(ctx).Where("bot_id = ?", botID).First(&l).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &l, err
}

func (s *Store) Save(ctx context.Context, l *License) error {
	return s.db.WithContext(ctx).Save(l).Error
}

func (s *Store) Revoke(ctx context.Context, botID int64, reason string) error {
	return s.db.WithContext(ctx).Model(&License{}).
		Where("bot_id = ?", botID).
		Updates(map[string]any{"status": "revoked", "revoked_reason": reason}).Error
}

// Package store contains member-bot repositories.
// All methods depend only on ports.DB — no postgres imports here.
package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
)

type Store struct{ db ports.DB }

func New(db ports.DB) *Store { return &Store{db: db} }

func (s *Store) FindOwnerByTelegramID(ctx context.Context, id int64) (*models.Owner, error) {
	var o models.Owner
	err := s.db.Conn().WithContext(ctx).Where("telegram_id = ?", id).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
	return &o, err
}

func (s *Store) CreateOwner(ctx context.Context, o *models.Owner) error {
	return s.db.Conn().WithContext(ctx).Create(o).Error
}

func (s *Store) FindLockByChannelID(ctx context.Context, channelID int64) (*models.Lock, error) {
	var l models.Lock
	err := s.db.Conn().WithContext(ctx).
		Where("channel_id = ? AND status = ?", channelID, models.LockActive).First(&l).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
	return &l, err
}

func (s *Store) CreateLock(ctx context.Context, l *models.Lock) error {
	return s.db.Conn().WithContext(ctx).Create(l).Error
}

func (s *Store) ExpireLock(ctx context.Context, lockID any) error {
	return s.db.Conn().WithContext(ctx).Model(&models.Lock{}).
		Where("id = ?", lockID).Update("status", models.LockExpired).Error
}

func (s *Store) FindExpiredLocks(ctx context.Context) ([]models.Lock, error) {
	var locks []models.Lock
	err := s.db.Conn().WithContext(ctx).Preload("Owner").
		Where("status = ? AND (expires_at < ? OR (max_members > 0 AND current_count >= max_members))",
			models.LockActive, time.Now()).Find(&locks).Error
	return locks, err
}

func (s *Store) FindActiveBots(ctx context.Context) ([]models.CheckBot, error) {
	var bots []models.CheckBot
	err := s.db.Conn().WithContext(ctx).Preload("Memberships").
		Where("is_active = true").Find(&bots).Error
	return bots, err
}

func (s *Store) CreateCheckBot(ctx context.Context, b *models.CheckBot) error {
	return s.db.Conn().WithContext(ctx).Create(b).Error
}

func (s *Store) AddBotMembership(ctx context.Context, m *models.BotChannelMembership) error {
	return s.db.Conn().WithContext(ctx).
		Where(models.BotChannelMembership{BotID: m.BotID, ChannelID: m.ChannelID}).
		Assign(models.BotChannelMembership{JoinedAt: m.JoinedAt, LastVerified: time.Now()}).
		FirstOrCreate(m).Error
}

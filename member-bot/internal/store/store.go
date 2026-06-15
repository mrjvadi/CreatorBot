// Package store contains member-bot repositories.
// All methods depend only on ports.DB — no postgres imports here.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
)

type Store struct{ db ports.DB }

func New(db ports.DB) *Store { return &Store{db: db} }

func (s *Store) FindOwnerByID(ctx context.Context, id uuid.UUID) (*models.Owner, error) {
	var o models.Owner
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
	return &o, err
}

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
	err := s.db.Conn().WithContext(ctx).
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

func (s *Store) FindLocksByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]models.Lock, error) {
	var locks []models.Lock
	err := s.db.Conn().WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("created_at DESC").Find(&locks).Error
	return locks, err
}

func (s *Store) DeleteLock(ctx context.Context, lockID uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.Lock{}, "id = ?", lockID).Error
}

func (s *Store) ListOwners(ctx context.Context) ([]models.Owner, error) {
	var owners []models.Owner
	err := s.db.Conn().WithContext(ctx).Order("created_at DESC").Find(&owners).Error
	return owners, err
}

func (s *Store) ListAllLocks(ctx context.Context) ([]models.Lock, error) {
	var locks []models.Lock
	err := s.db.Conn().WithContext(ctx).Order("created_at DESC").Find(&locks).Error
	return locks, err
}

func (s *Store) ApprovePayment(ctx context.Context, payID uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Payment{}).Where("id = ?", payID).
		Update("status", "confirmed").Error
}

func (s *Store) CreatePayment(ctx context.Context, p *models.Payment) error {
	return s.db.Conn().WithContext(ctx).Create(p).Error
}


func (s *Store) FindPendingPayments(ctx context.Context) ([]models.Payment, error) {
	var list []models.Payment
	return list, s.db.Conn().WithContext(ctx).Where("status = 'pending'").Find(&list).Error
}

func (s *Store) UpdateBalance(ctx context.Context, ownerID uuid.UUID, amount float64) error {
	return s.db.Conn().WithContext(ctx).Model(&models.Owner{}).
		Where("id = ?", ownerID).
		UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
}

func (s *Store) ClearBotMemberships(ctx context.Context) error {
	return s.db.Conn().WithContext(ctx).
		Where("1 = 1").Delete(&models.BotChannelMembership{}).Error
}

func (s *Store) DeactivateBotByID(ctx context.Context, botIDStr string) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.CheckBot{}).
		Where("id::text = ?", botIDStr).
		Update("is_active", false).Error
}

// DeleteBotByID یک check-bot را از DB حذف می‌کند.
func (s *Store) DeleteBotByID(ctx context.Context, botIDStr string) error {
	return s.db.Conn().WithContext(ctx).
		Where("id = ?", botIDStr).
		Delete(&models.CheckBot{}).Error
}

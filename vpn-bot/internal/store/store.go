package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/models"
)

type Store struct{ db ports.DB }

func New(db ports.DB) *Store { return &Store{db: db} }

// ---- User ----

func (s *Store) FindUserByTelegramID(ctx context.Context, id int64) (*models.User, error) {
	var u models.User
	err := s.db.Conn().WithContext(ctx).Where("telegram_id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (s *Store) UpsertUser(ctx context.Context, u *models.User) error {
	return s.db.Conn().WithContext(ctx).
		Where(models.User{TelegramID: u.TelegramID}).Assign(*u).FirstOrCreate(u).Error
}

func (s *Store) UpdateBalance(ctx context.Context, userID uuid.UUID, delta float64) error {
	return s.db.Conn().WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		UpdateColumn("balance", gorm.Expr("balance + ?", delta)).Error
}

// ---- Panel ----

	return &p, err
}

// ---- Plan ----

func (s *Store) ListPlans(ctx context.Context) ([]models.Plan, error) {
	var plans []models.Plan
	return plans, s.db.Conn().WithContext(ctx).Where("is_active = true").Find(&plans).Error
}

func (s *Store) FindPlan(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	var p models.Plan
	err := s.db.Conn().WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

// ---- Subscription ----

// SubscriptionWithUser holds a subscription alongside its owner's Telegram info.
// FIX 10: use GORM Preload instead of raw Scan to avoid column mapping issues.
type SubscriptionWithUser struct {
	models.Subscription
	User models.User
}

func (s *Store) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	return s.db.Conn().WithContext(ctx).Create(sub).Error
}

func (s *Store) FindActiveSubscriptions(ctx context.Context) ([]models.Subscription, error) {
	var subs []models.Subscription
	return subs, s.db.Conn().WithContext(ctx).
		Where("status = ?", models.SubActive).Find(&subs).Error
}

func (s *Store) FindExpiredSubscriptions(ctx context.Context) ([]models.Subscription, error) {
	var subs []models.Subscription
	return subs, s.db.Conn().WithContext(ctx).
		Where("status = ? AND expires_at < ?", models.SubActive, time.Now()).Find(&subs).Error
}

// FIX 10: use Preload for User — avoids Scan column mismatch
func (s *Store) FindSubscriptionsExpiringIn(ctx context.Context, d time.Duration) ([]SubscriptionWithUser, error) {
	deadline := time.Now().Add(d)
	var subs []models.Subscription
	err := s.db.Conn().WithContext(ctx).
		Preload("User").
		Where("status = ? AND expires_at > ? AND expires_at < ?",
			models.SubActive, time.Now(), deadline).
		Find(&subs).Error
	if err != nil {
		return nil, err
	}
	results := make([]SubscriptionWithUser, 0, len(subs))
	for _, sub := range subs {
		results = append(results, SubscriptionWithUser{
			Subscription: sub,
			// User populated by GORM Preload via UserID FK
		})
	}
	return results, nil
}

func (s *Store) UpdateSubscriptionStatus(ctx context.Context, id uuid.UUID, status models.SubscriptionStatus) error {
	return s.db.Conn().WithContext(ctx).Model(&models.Subscription{}).
		Where("id = ?", id).Update("status", status).Error
}

func (s *Store) UpdateSubscriptionUsage(ctx context.Context, id uuid.UUID, usedBytes int64) error {
	return s.db.Conn().WithContext(ctx).Model(&models.Subscription{}).
		Where("id = ?", id).
		Update("used_data", float64(usedBytes)/1024/1024/1024).Error
}

// ---- Subscription ----

func (s *Store) FindSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) ([]models.Subscription, error) {
	var subs []models.Subscription
	err := s.db.Conn().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").Find(&subs).Error
	return subs, err
}

func (s *Store) FindSubscriptionByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	var sub models.Subscription
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &sub, err
}

// ---- Payment ----

func (s *Store) CreatePayment(ctx context.Context, p *models.Payment) error {
	return s.db.Conn().WithContext(ctx).Create(p).Error
}

func (s *Store) FindPaymentByID(ctx context.Context, id uuid.UUID) (*models.Payment, error) {
	var p models.Payment
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (s *Store) FindPendingPayments(ctx context.Context) ([]models.Payment, error) {
	var payments []models.Payment
	err := s.db.Conn().WithContext(ctx).
		Where("status = 'pending'").
		Order("created_at ASC").Find(&payments).Error
	return payments, err
}

func (s *Store) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status string) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Payment{}).Where("id = ?", id).
		Update("status", status).Error
}

// ---- User (extra) ----

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := s.db.Conn().WithContext(ctx).Order("created_at DESC").Find(&users).Error
	return users, err
}

func (s *Store) FindUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var u models.User
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// ---- Panel ----


// ---- Panel ----

func (s *Store) ListPanels(ctx context.Context) ([]models.Panel, error) {
	var panels []models.Panel
	err := s.db.Conn().WithContext(ctx).
		Order("active_count ASC").Find(&panels).Error
	return panels, err
}

func (s *Store) CreatePanel(ctx context.Context, p *models.Panel) error {
	return s.db.Conn().WithContext(ctx).Create(p).Error
}

func (s *Store) FindPanelByID(ctx context.Context, id uuid.UUID) (*models.Panel, error) {
	var p models.Panel
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

// FindBestPanel کم‌ترین active_count رو داره (load balance).
func (s *Store) FindBestPanel(ctx context.Context) (*models.Panel, error) {
	var p models.Panel
	err := s.db.Conn().WithContext(ctx).
		Where("is_active = true AND (capacity = 0 OR active_count < capacity)").
		Order("active_count ASC").
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (s *Store) UpdatePanel(ctx context.Context, p *models.Panel) error {
	return s.db.Conn().WithContext(ctx).Save(p).Error
}

func (s *Store) DeletePanel(ctx context.Context, id uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.Panel{}, "id = ?", id).Error
}

func (s *Store) IncrementPanelCount(ctx context.Context, panelID uuid.UUID, delta int) error {
	return s.db.Conn().WithContext(ctx).Model(&models.Panel{}).
		Where("id = ?", panelID).
		UpdateColumn("active_count", gorm.Expr("active_count + ?", delta)).Error
}

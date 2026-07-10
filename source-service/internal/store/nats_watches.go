// This file holds persistence for NATS-triggered watches ("if a NATS
// message arrives, send it to a Telegram target"), parallel to watches.go
// (the Telegram-channel-triggered version).
package store

import (
	"context"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/source-service/internal/models"
)

// CreateNatsWatch persists a new NATS watch rule.
func (s *Store) CreateNatsWatch(ctx context.Context, w *models.NatsWatch) error {
	return s.db.Conn().WithContext(ctx).Create(w).Error
}

// ListActiveNatsWatches returns every active NATS watch rule for one
// account's phone number — used to restore live subscriptions on startup.
func (s *Store) ListActiveNatsWatches(ctx context.Context, phone string) ([]models.NatsWatch, error) {
	var rows []models.NatsWatch
	err := s.db.Conn().WithContext(ctx).Where("phone = ? AND active = ?", phone, true).Find(&rows).Error
	return rows, err
}

// DeactivateNatsWatch marks a NATS watch rule inactive (soft delete).
func (s *Store) DeactivateNatsWatch(ctx context.Context, id uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.NatsWatch{}).
		Where("id = ?", id).
		Update("active", false).Error
}

// This file holds persistence for real-time channel watches ("if source
// posts, forward to dest"), kept separate from the file registry
// (store.go) and other concerns per-file.
package store

import (
	"context"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/source-service/internal/models"
)

// CreateChannelWatch persists a new watch rule.
func (s *Store) CreateChannelWatch(ctx context.Context, w *models.ChannelWatch) error {
	return s.db.Conn().WithContext(ctx).Create(w).Error
}

// ListActiveChannelWatches returns every active watch rule for one
// account's phone number — used to restore in-memory watches on startup.
func (s *Store) ListActiveChannelWatches(ctx context.Context, phone string) ([]models.ChannelWatch, error) {
	var rows []models.ChannelWatch
	err := s.db.Conn().WithContext(ctx).Where("phone = ? AND active = ?", phone, true).Find(&rows).Error
	return rows, err
}

// DeactivateChannelWatch marks a watch rule inactive (soft delete).
func (s *Store) DeactivateChannelWatch(ctx context.Context, id uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.ChannelWatch{}).
		Where("id = ?", id).
		Update("active", false).Error
}

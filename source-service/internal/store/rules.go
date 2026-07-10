// This file holds persistence for generic rules (internal/rules): the
// trigger+condition+action definitions behind the create_rule task,
// parallel to watches.go and nats_watches.go but generic instead of one
// hardcoded combination.
package store

import (
	"context"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/source-service/internal/models"
)

// CreateRule persists a new rule.
func (s *Store) CreateRule(ctx context.Context, r *models.Rule) error {
	return s.db.Conn().WithContext(ctx).Create(r).Error
}

// ListActiveRules returns every active rule for one account's phone number
// — used to restore live triggers on startup.
func (s *Store) ListActiveRules(ctx context.Context, phone string) ([]models.Rule, error) {
	var rows []models.Rule
	err := s.db.Conn().WithContext(ctx).Where("phone = ? AND active = ?", phone, true).Find(&rows).Error
	return rows, err
}

// DeactivateRule marks a rule inactive (soft delete).
func (s *Store) DeactivateRule(ctx context.Context, id uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Rule{}).
		Where("id = ?", id).
		Update("active", false).Error
}

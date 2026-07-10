package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

func (s *Store) LogAudit(ctx context.Context, entry *models.AuditLog) error {
	entry.ID = newID()
	entry.InstanceID = s.instanceID
	entry.CreatedAt = time.Now()
	_, err := s.col(colAuditLogs).InsertOne(ctx, entry)
	return err
}

func (s *Store) ListAuditLogs(ctx context.Context, targetID string, pg, ps int) ([]models.AuditLog, error) {
	filter := s.f()
	if targetID != "" {
		filter = s.f(bson.E{Key: "target_id", Value: targetID})
	}
	var list []models.AuditLog
	err := s.col(colAuditLogs).Find(ctx, filter, &list,
		sortDesc("created_at"),
		skip(int64((pg-1)*ps)),
		limit(int64(ps)),
	)
	return list, err
}

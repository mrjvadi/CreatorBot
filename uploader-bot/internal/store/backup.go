package store

import (
	"context"
	"time"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

func (s *Store) CreateBackup(ctx context.Context, b *models.Backup) error {
	if b.ID == "" {
		b.ID = newID()
	}
	b.InstanceID = s.instanceID
	b.CreatedAt = time.Now()
	b.UpdatedAt = time.Now()
	_, err := s.col(colBackups).InsertOne(ctx, b)
	return err
}

func (s *Store) ListBackups(ctx context.Context, limit int) ([]models.Backup, error) {
	var backups []models.Backup
	err := s.col(colBackups).Find(ctx, s.f(), &backups,
		ports_sortDesc("created_at"), ports_limit(int64(limit)))
	return backups, err
}

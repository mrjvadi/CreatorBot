package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

func (s *Store) CreateFolder(ctx context.Context, f *models.Folder) error {
	if f.ID == "" {
		f.ID = newID()
	}
	f.InstanceID = s.instanceID
	f.CreatedAt = time.Now()
	f.UpdatedAt = time.Now()
	_, err := s.col(colFolders).InsertOne(ctx, f)
	return err
}

// ListFolders زیرپوشه‌های یک پوشه را برمی‌گرداند. parentID خالی = ریشه.
func (s *Store) ListFolders(ctx context.Context, parentID string) ([]models.Folder, error) {
	filter := s.f(
		bson.E{Key: "is_active", Value: true},
		bson.E{Key: "parent_id", Value: parentID},
	)
	var folders []models.Folder
	err := s.col(colFolders).Find(ctx, filter, &folders, ports_sortAsc("sort_order"))
	return folders, err
}

func (s *Store) DeleteFolder(ctx context.Context, id string) error {
	return s.col(colFolders).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: id}))
}

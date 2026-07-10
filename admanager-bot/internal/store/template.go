package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

func (s *Store) CreateTemplate(ctx context.Context, t *models.CampaignTemplate) error {
	t.ID = newID()
	t.InstanceID = s.instanceID
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	_, err := s.col(colTemplates).InsertOne(ctx, t)
	return err
}

func (s *Store) FindTemplate(ctx context.Context, id string) (*models.CampaignTemplate, error) {
	var t models.CampaignTemplate
	err := s.col(colTemplates).FindOne(ctx, s.f(bson.E{Key: "_id", Value: id}), &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) ListTemplates(ctx context.Context) ([]models.CampaignTemplate, error) {
	var list []models.CampaignTemplate
	err := s.col(colTemplates).Find(ctx, s.f(), &list, sortAsc("name"))
	return list, err
}

func (s *Store) DeleteTemplate(ctx context.Context, id string) error {
	return s.col(colTemplates).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: id}))
}

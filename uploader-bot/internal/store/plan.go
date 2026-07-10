package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

func (s *Store) CreateSubPlan(ctx context.Context, p *models.SubPlan) error {
	if p.ID == "" {
		p.ID = newID()
	}
	p.InstanceID = s.instanceID
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	_, err := s.col(colSubPlans).InsertOne(ctx, p)
	return err
}

func (s *Store) ListSubPlans(ctx context.Context) ([]models.SubPlan, error) {
	var plans []models.SubPlan
	err := s.col(colSubPlans).Find(ctx,
		s.f(bson.E{Key: "is_active", Value: true}), &plans,
		ports_sortAsc("sort_order"))
	return plans, err
}

func (s *Store) DeleteSubPlan(ctx context.Context, id string) error {
	return s.col(colSubPlans).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: id}))
}

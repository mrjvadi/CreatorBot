package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

func (s *Store) AddAd(ctx context.Context, ad *models.Ad) error {
	if ad.ID == "" {
		ad.ID = newID()
	}
	ad.InstanceID = s.instanceID
	ad.CreatedAt = time.Now()
	ad.UpdatedAt = time.Now()
	_, err := s.col(colAds).InsertOne(ctx, ad)
	return err
}

func (s *Store) ListAds(ctx context.Context) ([]models.Ad, error) {
	var ads []models.Ad
	err := s.col(colAds).Find(ctx,
		s.f(bson.E{Key: "is_active", Value: true}), &ads,
		ports_sortAsc("sort_order"))
	return ads, err
}

func (s *Store) RemoveAd(ctx context.Context, id string) error {
	return s.col(colAds).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: id}))
}

package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

func (s *Store) CreatePayment(ctx context.Context, p *models.Payment) error {
	if p.ID == "" {
		p.ID = newID()
	}
	p.InstanceID = s.instanceID
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if p.Status == "" {
		p.Status = models.PaymentPending
	}
	_, err := s.col(colPayments).InsertOne(ctx, p)
	return err
}

func (s *Store) ConfirmPayment(ctx context.Context, id string) error {
	now := time.Now()
	return s.col(colPayments).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "status", Value: models.PaymentConfirmed},
			{Key: "confirmed_at", Value: now},
		}))
}

// RejectPayment یک پرداخت در انتظار (کارت/TON/TRON دستی) را رد می‌کند.
func (s *Store) RejectPayment(ctx context.Context, id string) error {
	return s.col(colPayments).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "status", Value: models.PaymentFailed}}))
}

func (s *Store) FindPayment(ctx context.Context, id string) (*models.Payment, error) {
	var p models.Payment
	err := s.col(colPayments).FindOne(ctx, s.f(bson.E{Key: "_id", Value: id}), &p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) FindPaymentByAuthority(ctx context.Context, authority string) (*models.Payment, error) {
	var p models.Payment
	err := s.col(colPayments).FindOne(ctx, s.f(bson.E{Key: "authority", Value: authority}), &p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

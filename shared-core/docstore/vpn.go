package docstore

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/shared-core/documents"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// VPNSubscriptionStore مدیریت اشتراک‌های VPN.
type VPNSubscriptionStore struct {
	Base
}

func NewVPNSubscriptionStore(ds ports.DocumentStore, instanceID string) *VPNSubscriptionStore {
	return &VPNSubscriptionStore{Base: NewBase(ds, instanceID)}
}

func (s *VPNSubscriptionStore) Create(ctx context.Context, sub *documents.VPNSubscription) error {
	sub.DocBase = s.newDocBase()
	_, err := s.col("subscriptions").InsertOne(ctx, sub)
	return err
}

func (s *VPNSubscriptionStore) FindByUserID(ctx context.Context, userID int64) (*documents.VPNSubscription, error) {
	var sub documents.VPNSubscription
	filter := s.baseFilter(
		bson.E{Key: "user_id", Value: userID},
		bson.E{Key: "status", Value: "active"},
	)
	err := s.col("subscriptions").FindOne(ctx, filter, &sub)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &sub, err
}

func (s *VPNSubscriptionStore) FindExpiring(ctx context.Context, within time.Duration) ([]documents.VPNSubscription, error) {
	var subs []documents.VPNSubscription
	deadline := time.Now().Add(within)
	filter := s.baseFilter(
		bson.E{Key: "status", Value: "active"},
		bson.E{Key: "expires_at", Value: bson.D{
			{Key: "$gt", Value: time.Now()},
			{Key: "$lt", Value: deadline},
		}},
	)
	err := s.col("subscriptions").Find(ctx, filter, &subs)
	return subs, err
}

func (s *VPNSubscriptionStore) FindExpired(ctx context.Context) ([]documents.VPNSubscription, error) {
	var subs []documents.VPNSubscription
	filter := s.baseFilter(
		bson.E{Key: "status", Value: "active"},
		bson.E{Key: "expires_at", Value: bson.D{{Key: "$lt", Value: time.Now()}}},
	)
	err := s.col("subscriptions").Find(ctx, filter, &subs)
	return subs, err
}

func (s *VPNSubscriptionStore) UpdateStatus(ctx context.Context, id string, status string) error {
	filter := s.baseFilter(bson.E{Key: "_id", Value: id})
	return s.col("subscriptions").UpdateOne(ctx, filter,
		setUpdate(bson.D{{Key: "status", Value: status}}))
}

func (s *VPNSubscriptionStore) UpdateUsage(ctx context.Context, id string, usedGB float64) error {
	filter := s.baseFilter(bson.E{Key: "_id", Value: id})
	return s.col("subscriptions").UpdateOne(ctx, filter,
		setUpdate(bson.D{{Key: "used_data_gb", Value: usedGB}}))
}

// VPNPaymentStore مدیریت پرداخت‌های VPN.
type VPNPaymentStore struct {
	Base
}

func NewVPNPaymentStore(ds ports.DocumentStore, instanceID string) *VPNPaymentStore {
	return &VPNPaymentStore{Base: NewBase(ds, instanceID)}
}

func (s *VPNPaymentStore) Create(ctx context.Context, p *documents.VPNPaymentReceipt) error {
	p.DocBase = s.newDocBase()
	_, err := s.col("vpn_payments").InsertOne(ctx, p)
	return err
}

func (s *VPNPaymentStore) FindPending(ctx context.Context) ([]documents.VPNPaymentReceipt, error) {
	var payments []documents.VPNPaymentReceipt
	filter := s.baseFilter(bson.E{Key: "status", Value: "pending"})
	err := s.col("vpn_payments").Find(ctx, filter, &payments,
		ports.WithSort(bson.D{{Key: "created_at", Value: -1}}))
	return payments, err
}

func (s *VPNPaymentStore) Confirm(ctx context.Context, id string) error {
	filter := s.baseFilter(bson.E{Key: "_id", Value: id})
	now := time.Now()
	return s.col("vpn_payments").UpdateOne(ctx, filter,
		setUpdate(bson.D{
			{Key: "status", Value: "done"},
			{Key: "paid_at", Value: now},
		}))
}

package docstore

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/shared-core/documents"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// MemberVerificationStore نتایج چک عضویت.
type MemberVerificationStore struct {
	Base
}

func NewMemberVerificationStore(ds ports.DocumentStore, instanceID string) *MemberVerificationStore {
	return &MemberVerificationStore{Base: NewBase(ds, instanceID)}
}

func (s *MemberVerificationStore) Create(ctx context.Context, v *documents.MemberVerification) error {
	v.DocBase = s.newDocBase()
	v.CheckedAt = time.Now()
	_, err := s.col("member_verifications").InsertOne(ctx, v)
	return err
}

// LastCheck آخرین نتیجه چک برای یک user در یک lock.
func (s *MemberVerificationStore) LastCheck(ctx context.Context, lockID string, userID int64) (*documents.MemberVerification, error) {
	var v documents.MemberVerification
	filter := s.baseFilter(
		bson.E{Key: "lock_id", Value: lockID},
		bson.E{Key: "user_id", Value: userID},
	)
	err := s.col("member_verifications").FindOne(ctx, filter, &v)
	if err != nil {
		return nil, nil
	}
	return &v, nil
}

// CountToday تعداد چک‌های امروز برای یک lock.
func (s *MemberVerificationStore) CountToday(ctx context.Context, lockID string) (int64, error) {
	startOfDay := time.Now().Truncate(24 * time.Hour)
	filter := s.baseFilter(
		bson.E{Key: "lock_id", Value: lockID},
		bson.E{Key: "checked_at", Value: bson.D{{Key: "$gte", Value: startOfDay}}},
	)
	return s.col("member_verifications").CountDocuments(ctx, filter)
}

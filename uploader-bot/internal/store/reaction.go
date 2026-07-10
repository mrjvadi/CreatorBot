package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

const colReactions = "reactions"

// SetReaction واکنش کاربر را تنظیم می‌کند. اگر همان واکنش قبلاً ثبت شده باشد،
// حذف می‌شود (toggle). خروجی: مقدار نهاییِ واکنش کاربر (0 یعنی برداشته شد).
func (s *Store) SetReaction(ctx context.Context, code string, uid int64, val int) int {
	filter := s.f(bson.E{Key: "code", Value: code}, bson.E{Key: "user_id", Value: uid})
	var existing models.Reaction
	err := s.col(colReactions).FindOne(ctx, filter, &existing)
	if errors.Is(err, mongo.ErrNoDocuments) {
		r := &models.Reaction{Code: code, UserID: uid, Value: val}
		r.ID = newID()
		r.InstanceID = s.instanceID
		r.CreatedAt = time.Now()
		r.UpdatedAt = time.Now()
		_, insErr := s.col(colReactions).InsertOne(ctx, r)
		s.logErr("SetReaction: insert", insErr)
		return val
	}
	if err != nil {
		s.logErr("SetReaction: find", err)
		return 0
	}
	if existing.Value == val {
		// همان واکنش دوباره → برداشتن
		s.logErr("SetReaction: delete", s.col(colReactions).DeleteOne(ctx, filter))
		return 0
	}
	s.logErr("SetReaction: update", s.col(colReactions).UpdateOne(ctx, filter, set(bson.D{{Key: "value", Value: val}})))
	return val
}

// CountReactions تعداد لایک و دیسلایک واقعی یک کد را برمی‌گرداند.
func (s *Store) CountReactions(ctx context.Context, code string) (likes, dislikes int64) {
	var err error
	likes, err = s.col(colReactions).CountDocuments(ctx,
		s.f(bson.E{Key: "code", Value: code}, bson.E{Key: "value", Value: 1}))
	s.logErr("CountReactions: likes", err)
	dislikes, err = s.col(colReactions).CountDocuments(ctx,
		s.f(bson.E{Key: "code", Value: code}, bson.E{Key: "value", Value: -1}))
	s.logErr("CountReactions: dislikes", err)
	return
}

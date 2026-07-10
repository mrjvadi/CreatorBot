package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const (
	collLogs   = "log_entries"
	collTopics = "log_topics"
)

// Store لایه‌ی دسترسی MongoDB این سرویس.
type Store struct {
	db ports.DocumentStore
}

func New(db ports.DocumentStore) *Store { return &Store{db: db} }

// EnsureIndexes ایندکس‌های لازم برای کوئری سریع را می‌سازد — صدا زدن این در
// startup ایمن است (اگر از قبل باشند، خطا نادیده گرفته می‌شود).
func (s *Store) EnsureIndexes(ctx context.Context) {
	_ = s.db.Collection(collLogs).CreateIndex(ctx, bson.D{{Key: "timestamp", Value: -1}}, false)
	_ = s.db.Collection(collLogs).CreateIndex(ctx, bson.D{{Key: "service", Value: 1}}, false)
	_ = s.db.Collection(collLogs).CreateIndex(ctx, bson.D{{Key: "level", Value: 1}}, false)
	_ = s.db.Collection(collTopics).CreateIndex(ctx, bson.D{{Key: "service", Value: 1}}, true)
}

func (s *Store) SaveLog(ctx context.Context, e *LogEntry) error {
	e.ReceivedAt = time.Now()
	_, err := s.db.Collection(collLogs).InsertOne(ctx, e)
	return err
}

// QueryFilter پارامترهای کوئری لاگ‌ها — همه اختیاری‌اند.
type QueryFilter struct {
	Service string
	Level   string
	Query   string // جست‌وجوی متنی در message (regex، case-insensitive)
	From    *time.Time
	To      *time.Time
	Limit   int64
	Skip    int64
}

// QueryLogs لاگ‌ها را بر اساس فیلتر برمی‌گرداند، جدیدترین اول.
func (s *Store) QueryLogs(ctx context.Context, f QueryFilter) ([]LogEntry, error) {
	filter := bson.M{}
	if f.Service != "" {
		filter["service"] = f.Service
	}
	if f.Level != "" {
		filter["level"] = f.Level
	}
	if f.Query != "" {
		filter["message"] = bson.M{"$regex": f.Query, "$options": "i"}
	}
	if f.From != nil || f.To != nil {
		ts := bson.M{}
		if f.From != nil {
			ts["$gte"] = *f.From
		}
		if f.To != nil {
			ts["$lte"] = *f.To
		}
		filter["timestamp"] = ts
	}

	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var results []LogEntry
	err := s.db.Collection(collLogs).Find(ctx, filter, &results,
		ports.WithLimit(limit),
		ports.WithSkip(f.Skip),
		ports.WithSort(bson.D{{Key: "timestamp", Value: -1}}),
	)
	return results, err
}

// GetTopicID شناسه‌ی topic تلگرام یک سرویس را برمی‌گرداند (اگر قبلاً ساخته شده).
func (s *Store) GetTopicID(ctx context.Context, service string) (int, bool) {
	var m TopicMapping
	err := s.db.Collection(collTopics).FindOne(ctx, bson.M{"service": service}, &m)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, false
		}
		return 0, false
	}
	return m.MessageThreadID, true
}

// SaveTopicID نگاشت سرویس→topic را ذخیره می‌کند.
func (s *Store) SaveTopicID(ctx context.Context, service string, threadID int) error {
	_, err := s.db.Collection(collTopics).InsertOne(ctx, TopicMapping{Service: service, MessageThreadID: threadID})
	return err
}

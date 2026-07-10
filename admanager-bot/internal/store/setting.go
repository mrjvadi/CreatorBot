package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

// GetSettings تنظیمات ربات را برمی‌گرداند — اگر نبود پیش‌فرض می‌سازد.
func (s *Store) GetSettings(ctx context.Context) (*models.BotSettings, error) {
	var st models.BotSettings
	err := s.col(colSettings).FindOne(ctx, bson.D{{Key: "_id", Value: s.instanceID}}, &st)
	if err != nil {
		// اولین بار — مقادیر پیش‌فرض
		st = models.BotSettings{
			InstanceID:            s.instanceID,
			DefaultStartHour:      models.DefaultStartHour,
			DefaultEndHour:        models.DefaultEndHour,
			ReminderMinutesBefore: models.DefaultReminderMinutesBefore,
			UpdatedAt:             time.Now(),
		}
		_, _ = s.col(colSettings).InsertOne(ctx, &st)
	}
	return &st, nil
}

func (s *Store) UpdateSettings(ctx context.Context, fields bson.D) error {
	fields = append(fields, bson.E{Key: "updated_at", Value: time.Now()})
	// GetSettings اطمینان می‌دهد سند حتماً قبلاً درج شده
	return s.col(colSettings).UpdateOne(ctx,
		bson.D{{Key: "_id", Value: s.instanceID}},
		bson.D{{Key: "$set", Value: fields}},
	)
}

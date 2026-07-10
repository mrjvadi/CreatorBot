package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// LogDownload شمارش دانلود یک کاربر برای یک کد را یک واحد زیاد می‌کند.
func (s *Store) LogDownload(ctx context.Context, userID, codeID string) error {
	filter := s.f(bson.E{Key: "user_id", Value: userID}, bson.E{Key: "code_id", Value: codeID})
	var log models.DownloadLog
	err := s.col(colDownloads).FindOne(ctx, filter, &log)
	if errors.Is(err, mongo.ErrNoDocuments) {
		dl := &models.DownloadLog{UserID: userID, CodeID: codeID, Count: 1}
		dl.ID = newID()
		dl.InstanceID = s.instanceID
		dl.CreatedAt = time.Now()
		dl.UpdatedAt = time.Now()
		_, e := s.col(colDownloads).InsertOne(ctx, dl)
		return e
	}
	if err != nil {
		return err
	}
	return s.col(colDownloads).UpdateOne(ctx, filter,
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "count", Value: 1}}},
			{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now()}}},
		})
}

// DeleteUserDownloads همه‌ی لاگ‌های دانلود یک کاربر را حذف می‌کند.
func (s *Store) DeleteUserDownloads(ctx context.Context, userID string) error {
	var logs []models.DownloadLog
	if err := s.col(colDownloads).Find(ctx,
		s.f(bson.E{Key: "user_id", Value: userID}), &logs); err != nil {
		return err
	}
	for _, l := range logs {
		s.logErr("DeleteUserDownloads", s.col(colDownloads).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: l.ID})))
	}
	return nil
}

func (s *Store) GetDownloadCount(ctx context.Context, userID, codeID string) int {
	var log models.DownloadLog
	err := s.col(colDownloads).FindOne(ctx,
		s.f(bson.E{Key: "user_id", Value: userID}, bson.E{Key: "code_id", Value: codeID}), &log)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			s.logErr("GetDownloadCount", err)
		}
		return 0
	}
	return log.Count
}

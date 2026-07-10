package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

func (s *Store) CreateFile(ctx context.Context, f *models.File) error {
	if f.ID == "" {
		f.ID = newID()
	}
	f.InstanceID = s.instanceID
	f.CreatedAt = time.Now()
	f.UpdatedAt = time.Now()
	_, err := s.col(colFiles).InsertOne(ctx, f)
	return err
}

// AddFileToCode شناسه‌ی فایل را به انتهای لیست فایل‌های کد اضافه می‌کند.
// order نگه داشته شده تا امضای متد سازگار بماند؛ ترتیب بر اساس افزودن است.
func (s *Store) AddFileToCode(ctx context.Context, codeID, fileID string, order int) error {
	return s.col(colCodes).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: codeID}),
		bson.D{
			{Key: "$push", Value: bson.D{{Key: "file_ids", Value: fileID}}},
			{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now()}}},
		})
}

func (s *Store) RemoveFileFromCode(ctx context.Context, codeID, fileID string) error {
	return s.col(colCodes).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: codeID}),
		bson.D{
			{Key: "$pull", Value: bson.D{{Key: "file_ids", Value: fileID}}},
			{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now()}}},
		})
}

// SetFileStorage مرجع کانال ذخیره‌سازی یک فایل را تنظیم می‌کند.
func (s *Store) SetFileStorage(ctx context.Context, fileID string, chatID int64, msgID int) error {
	return s.col(colFiles).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: fileID}),
		set(bson.D{
			{Key: "storage_chat_id", Value: chatID},
			{Key: "storage_msg_id", Value: msgID},
		}))
}

// SetFileThumbnail تامبنیل/کاور یک فایل را تنظیم می‌کند.
func (s *Store) SetFileThumbnail(ctx context.Context, fileID, thumb string) error {
	return s.col(colFiles).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: fileID}),
		set(bson.D{{Key: "thumbnail", Value: thumb}}))
}

// GetFilesForCode فایل‌های یک کد را به ترتیب درست برمی‌گرداند.
func (s *Store) GetFilesForCode(ctx context.Context, codeID string) ([]models.File, error) {
	code, err := s.FindCodeByID(ctx, codeID)
	if err != nil || code == nil || len(code.FileIDs) == 0 {
		return nil, err
	}
	var files []models.File
	filter := s.f(bson.E{Key: "_id", Value: bson.D{{Key: "$in", Value: code.FileIDs}}})
	if err := s.col(colFiles).Find(ctx, filter, &files); err != nil {
		return nil, err
	}
	// مرتب‌سازی بر اساس ترتیب FileIDs
	byID := make(map[string]models.File, len(files))
	for _, f := range files {
		byID[f.ID] = f
	}
	ordered := make([]models.File, 0, len(code.FileIDs))
	for _, id := range code.FileIDs {
		if f, ok := byID[id]; ok {
			ordered = append(ordered, f)
		}
	}
	return ordered, nil
}

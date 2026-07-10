package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// ListAllCodes همه‌ی کدهای این ربات را برمی‌گرداند (برای بکاپ/عملیات انبوه).
func (s *Store) ListAllCodes(ctx context.Context) ([]models.Code, error) {
	var codes []models.Code
	err := s.col(colCodes).Find(ctx, s.f(), &codes)
	return codes, err
}

// ListAllFiles همه‌ی فایل‌های این ربات را برمی‌گرداند.
func (s *Store) ListAllFiles(ctx context.Context) ([]models.File, error) {
	var files []models.File
	err := s.col(colFiles).Find(ctx, s.f(), &files)
	return files, err
}

// DeleteAllCodes همه‌ی کدها و فایل‌ها را حذف می‌کند.
func (s *Store) DeleteAllCodes(ctx context.Context) error {
	codes, err := s.ListAllCodes(ctx)
	if err != nil {
		return err
	}
	for _, c := range codes {
		s.InvalidateCode(ctx, c.Code)
		s.logErr("DeleteAllCodes: delete code", s.col(colCodes).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: c.ID})))
	}
	files, err := s.ListAllFiles(ctx)
	if err != nil {
		return err
	}
	for _, f := range files {
		s.logErr("DeleteAllCodes: delete file", s.col(colFiles).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: f.ID})))
	}
	return nil
}

// SetForwardLockAll قفل فوروارد را روی همه‌ی کدها اعمال/برمی‌دارد.
func (s *Store) SetForwardLockAll(ctx context.Context, on bool) (int, error) {
	codes, err := s.ListAllCodes(ctx)
	if err != nil {
		return 0, err
	}
	for _, c := range codes {
		s.logErr("SetForwardLockAll", s.col(colCodes).UpdateOne(ctx, s.f(bson.E{Key: "_id", Value: c.ID}),
			set(bson.D{{Key: "forward_lock", Value: on}})))
		s.InvalidateCode(ctx, c.Code)
	}
	return len(codes), nil
}

// SetAutoDeleteAll زمان حذف خودکار را روی همه‌ی کدها اعمال می‌کند (0=خاموش).
func (s *Store) SetAutoDeleteAll(ctx context.Context, sec int) (int, error) {
	codes, err := s.ListAllCodes(ctx)
	if err != nil {
		return 0, err
	}
	for _, c := range codes {
		s.logErr("SetAutoDeleteAll", s.col(colCodes).UpdateOne(ctx, s.f(bson.E{Key: "_id", Value: c.ID}),
			set(bson.D{{Key: "auto_delete", Value: sec}})))
		s.InvalidateCode(ctx, c.Code)
	}
	return len(codes), nil
}

// InsertCodeRaw یک کد را با حفظ فیلدها درج می‌کند (برای ریستور).
func (s *Store) InsertCodeRaw(ctx context.Context, c *models.Code) error {
	c.InstanceID = s.instanceID
	if c.ID == "" {
		c.ID = newID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	c.UpdatedAt = time.Now()
	_, err := s.col(colCodes).InsertOne(ctx, c)
	return err
}

// InsertFileRaw یک فایل را با حفظ فیلدها درج می‌کند (برای ریستور).
func (s *Store) InsertFileRaw(ctx context.Context, f *models.File) error {
	f.InstanceID = s.instanceID
	if f.ID == "" {
		f.ID = newID()
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now()
	}
	f.UpdatedAt = time.Now()
	_, err := s.col(colFiles).InsertOne(ctx, f)
	return err
}

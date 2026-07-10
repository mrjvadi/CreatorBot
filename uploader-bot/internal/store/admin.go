package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

func (s *Store) IsAdmin(ctx context.Context, tgID int64) bool {
	n, err := s.col(colAdmins).CountDocuments(ctx, s.f(bson.E{Key: "telegram_id", Value: tgID}))
	s.logErr("IsAdmin", err)
	return n > 0
}

func (s *Store) AddAdmin(ctx context.Context, tgID int64, username string) error {
	if s.IsAdmin(ctx, tgID) {
		return nil
	}
	a := &models.Admin{TelegramID: tgID, Username: username}
	a.ID = newID()
	a.InstanceID = s.instanceID
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	_, err := s.col(colAdmins).InsertOne(ctx, a)
	return err
}

func (s *Store) RemoveAdmin(ctx context.Context, tgID int64) error {
	return s.col(colAdmins).DeleteOne(ctx,
		s.f(bson.E{Key: "telegram_id", Value: tgID}, bson.E{Key: "is_owner", Value: false}))
}

func (s *Store) ListAdmins(ctx context.Context) ([]models.Admin, error) {
	var admins []models.Admin
	err := s.col(colAdmins).Find(ctx, s.f(), &admins)
	return admins, err
}

// GetAdmin یک ادمین را با آیدی تلگرام برمی‌گرداند (nil اگر نباشد).
func (s *Store) GetAdmin(ctx context.Context, tgID int64) (*models.Admin, error) {
	var a models.Admin
	err := s.col(colAdmins).FindOne(ctx, s.f(bson.E{Key: "telegram_id", Value: tgID}), &a)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		// نکته: قبلاً این حالت &a (یک Admin خالی با Perms nil) را همراه با
		// خطا برمی‌گرداند. چون بیشتر فراخوان‌ها فقط err را نادیده می‌گیرند و
		// روی «a == nil» چک می‌کنند، یک قطعیِ گذرای Mongo باعث می‌شد ادمین با
		// دسترسیِ صفر تلقی شود (نه خطا). حالا صریحاً nil برمی‌گردد تا فرقی با
		// «واقعاً پیدا نشد» نداشته باشد ولی خطا هم گم نشود.
		return nil, err
	}
	return &a, nil
}

// SetAdminPerms دسترسی‌های یک ادمین را تنظیم می‌کند.
func (s *Store) SetAdminPerms(ctx context.Context, tgID int64, perms []string) error {
	return s.col(colAdmins).UpdateOne(ctx,
		s.f(bson.E{Key: "telegram_id", Value: tgID}),
		set(bson.D{{Key: "perms", Value: perms}}))
}

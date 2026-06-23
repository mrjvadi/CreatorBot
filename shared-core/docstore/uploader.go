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

// CodeStore مدیریت کدهای دریافت فایل.
type CodeStore struct {
	Base
}

func NewCodeStore(ds ports.DocumentStore, instanceID string) *CodeStore {
	return &CodeStore{Base: NewBase(ds, instanceID)}
}

func (s *CodeStore) Create(ctx context.Context, code *documents.Code) error {
	code.DocBase = s.newDocBase()
	_, err := s.col("codes").InsertOne(ctx, code)
	return err
}

func (s *CodeStore) FindByCode(ctx context.Context, codeStr string) (*documents.Code, error) {
	var c documents.Code
	err := s.col("codes").FindOne(ctx,
		s.baseFilter(bson.E{Key: "code", Value: codeStr}), &c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &c, err
}

func (s *CodeStore) IncrementUse(ctx context.Context, codeID string) error {
	filter := s.baseFilter(bson.E{Key: "_id", Value: codeID})
	update := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "used_count", Value: 1}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now()}}},
	}
	return s.col("codes").UpdateOne(ctx, filter, update)
}

// IsValid بررسی می‌کند کد هنوز معتبر است.
func (s *CodeStore) IsValid(c *documents.Code) bool {
	if c.ExpiresAt != nil && time.Now().After(*c.ExpiresAt) {
		return false
	}
	if c.Type == documents.CodeOnce && c.UsedCount >= 1 {
		return false
	}
	if c.Type == documents.CodeLimited && c.UsedCount >= c.MaxUse {
		return false
	}
	return true
}

// FileStore مدیریت فایل‌های آپلود شده.
type FileStore struct {
	Base
}

func NewFileStore(ds ports.DocumentStore, instanceID string) *FileStore {
	return &FileStore{Base: NewBase(ds, instanceID)}
}

func (s *FileStore) Create(ctx context.Context, file *documents.File) error {
	file.DocBase = s.newDocBase()
	_, err := s.col("files").InsertOne(ctx, file)
	return err
}

func (s *FileStore) FindByIDs(ctx context.Context, ids []string) ([]documents.File, error) {
	var files []documents.File
	filter := s.baseFilter(bson.E{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}})
	err := s.col("files").Find(ctx, filter, &files, ports.WithSort(bson.D{{Key: "created_at", Value: 1}}))
	return files, err
}

// BotUserStore کاربران ربات.
type BotUserStore struct {
	Base
}

func NewBotUserStore(ds ports.DocumentStore, instanceID string) *BotUserStore {
	return &BotUserStore{Base: NewBase(ds, instanceID)}
}

func (s *BotUserStore) Upsert(ctx context.Context, user *documents.BotUser) error {
	filter := s.baseFilter(bson.E{Key: "telegram_id", Value: user.TelegramID})
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "username", Value: user.Username},
			{Key: "first_name", Value: user.FirstName},
			{Key: "is_blocked", Value: user.IsBlocked},
			{Key: "updated_at", Value: time.Now()},
			{Key: "instance_id", Value: s.instanceID},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "telegram_id", Value: user.TelegramID},
			{Key: "created_at", Value: time.Now()},
		}},
	}
	return s.col("bot_users").UpdateOne(ctx, filter, update)
}

func (s *BotUserStore) FindByTelegramID(ctx context.Context, id int64) (*documents.BotUser, error) {
	var u documents.BotUser
	err := s.col("bot_users").FindOne(ctx,
		s.baseFilter(bson.E{Key: "telegram_id", Value: id}), &u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &u, err
}

func (s *BotUserStore) SetBlocked(ctx context.Context, telegramID int64, blocked bool) error {
	filter := s.baseFilter(bson.E{Key: "telegram_id", Value: telegramID})
	return s.col("bot_users").UpdateOne(ctx, filter,
		setUpdate(bson.D{{Key: "is_blocked", Value: blocked}}))
}

func (s *BotUserStore) Count(ctx context.Context) (int64, error) {
	return s.col("bot_users").CountDocuments(ctx, s.baseFilter())
}

// ── CodeStore — متدهای تکمیلی ─────────────────────────────

func (s *CodeStore) List(ctx context.Context, limit int) ([]documents.Code, error) {
	var codes []documents.Code
	opts := []ports.FindOption{ports.WithSort(bson.D{{Key: "created_at", Value: -1}})}
	if limit > 0 {
		opts = append(opts, ports.WithLimit(int64(limit)))
	}
	err := s.col("codes").Find(ctx, s.baseFilter(), &codes, opts...)
	return codes, err
}

func (s *CodeStore) DeleteByCode(ctx context.Context, code string) error {
	filter := s.baseFilter(bson.E{Key: "code", Value: code})
	return s.col("codes").DeleteOne(ctx, filter)
}

func (s *CodeStore) Count(ctx context.Context) (int64, error) {
	return s.col("codes").CountDocuments(ctx, s.baseFilter())
}

func (s *CodeStore) CountActive(ctx context.Context) (int64, error) {
	now := time.Now()
	filter := s.baseFilter(
		bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "expires_at", Value: nil}},
			bson.D{{Key: "expires_at", Value: bson.D{{Key: "$gt", Value: now}}}},
		}},
	)
	return s.col("codes").CountDocuments(ctx, filter)
}

// ── FileStore — متدهای تکمیلی ────────────────────────────

// CreateFromInfo یک فایل ارسالی از ادمین را ذخیره می‌کند.
// fileID از Telegram، fileType نوع فایل، caption متن همراه.
// شناسه‌ی رشته‌ای سند ساخته‌شده را برمی‌گرداند (یا "" در صورت خطا).
func (s *FileStore) CreateFromInfo(ctx context.Context, fileID, fileType, caption string, uploaderID int64) string {
	file := &documents.File{
		TelegramFileID: fileID,
		FileType:       fileType,
		Caption:        caption,
		UploaderID:     uploaderID,
	}
	file.DocBase = s.newDocBase()
	id, err := s.col("files").InsertOne(ctx, file)
	if err != nil {
		return ""
	}
	return id
}

func (s *FileStore) Count(ctx context.Context) (int64, error) {
	return s.col("files").CountDocuments(ctx, s.baseFilter())
}

// ── BotUserStore — متدهای تکمیلی ─────────────────────────

func (s *BotUserStore) List(ctx context.Context, limit int) ([]documents.BotUser, error) {
	var users []documents.BotUser
	opts := []ports.FindOption{ports.WithSort(bson.D{{Key: "created_at", Value: -1}})}
	if limit > 0 {
		opts = append(opts, ports.WithLimit(int64(limit)))
	}
	err := s.col("bot_users").Find(ctx, s.baseFilter(), &users, opts...)
	return users, err
}

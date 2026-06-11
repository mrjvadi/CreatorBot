// Package store contains repository types for uploader-bot.
// All methods accept ports.DB — never import postgres directly here.
// If you swap the DB adapter, this file needs zero changes.
package store

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// ---- CodeStore ----

type CodeStore struct{ db ports.DB }

func NewCodeStore(db ports.DB) *CodeStore { return &CodeStore{db: db} }

func (s *CodeStore) FindByCode(ctx context.Context, code string) (*models.Code, error) {
	var c models.Code
	err := s.db.Conn().WithContext(ctx).
		Preload("Files").
		Where("code = ?", code).
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (s *CodeStore) Create(ctx context.Context, c *models.Code) error {
	return s.db.Conn().WithContext(ctx).Create(c).Error
}

func (s *CodeStore) IncrementUse(ctx context.Context, id any) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Code{}).
		Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

// ---- FileStore ----

type FileStore struct{ db ports.DB }

func NewFileStore(db ports.DB) *FileStore { return &FileStore{db: db} }

func (s *FileStore) Create(ctx context.Context, f *models.File) error {
	return s.db.Conn().WithContext(ctx).Create(f).Error
}

// ---- SettingStore ----

type SettingStore struct{ db ports.DB }

func NewSettingStore(db ports.DB) *SettingStore { return &SettingStore{db: db} }

func (s *SettingStore) Get(ctx context.Context, key string) (string, error) {
	var setting models.Setting
	err := s.db.Conn().WithContext(ctx).Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return setting.Value, err
}

func (s *SettingStore) Set(ctx context.Context, key, value string) error {
	return s.db.Conn().WithContext(ctx).
		Where(models.Setting{Key: key}).
		Assign(models.Setting{Value: value}).
		FirstOrCreate(&models.Setting{}).Error
}

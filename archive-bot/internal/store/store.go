package store

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/archive-bot/internal/models"
)

type Store struct{ db ports.DB }

func New(db ports.DB) *Store { return &Store{db: db} }

func (s *Store) FindUserByTelegramID(ctx context.Context, id int64) (*models.User, error) {
	var u models.User
	err := s.db.Conn().WithContext(ctx).Where("telegram_id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
	return &u, err
}

func (s *Store) UpsertUser(ctx context.Context, u *models.User) error {
	return s.db.Conn().WithContext(ctx).
		Where(models.User{TelegramID: u.TelegramID}).Assign(*u).FirstOrCreate(u).Error
}

func (s *Store) CreateFile(ctx context.Context, f *models.File) error {
	return s.db.Conn().WithContext(ctx).Create(f).Error
}

func (s *Store) ListCategories(ctx context.Context) ([]models.Category, error) {
	var cats []models.Category
	return cats, s.db.Conn().WithContext(ctx).Find(&cats).Error
}

func (s *Store) FindOrCreateCategory(ctx context.Context, name string) (*models.Category, error) {
	var cat models.Category
	err := s.db.Conn().WithContext(ctx).
		Where(models.Category{Name: name}).FirstOrCreate(&cat).Error
	return &cat, err
}

func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var s2 models.Setting
	err := s.db.Conn().WithContext(ctx).Where("key = ?", key).First(&s2).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return "", nil }
	return s2.Value, err
}

func (s *Store) UpsertUserByID(ctx context.Context, telegramID int64, username, firstName string) {
	u := &models.User{TelegramID: telegramID, Username: username, FirstName: firstName}
	s.db.Conn().WithContext(ctx).
		Where(models.User{TelegramID: telegramID}).
		Assign(*u).FirstOrCreate(u)
}

func (s *Store) FindFilesByCategory(ctx context.Context, catIDStr string) ([]models.File, error) {
	var files []models.File
	err := s.db.Conn().WithContext(ctx).
		Where("category_id = ?", catIDStr).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

func (s *Store) FindCategoryByID(ctx context.Context, idStr string) (*models.Category, error) {
	var cat models.Category
	err := s.db.Conn().WithContext(ctx).Where("id = ?", idStr).First(&cat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cat, err
}

func (s *Store) DeleteFile(ctx context.Context, idStr string) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.File{}, "id = ?", idStr).Error
}

func (s *Store) FindFileByID(ctx context.Context, idStr string) (*models.File, error) {
	var f models.File
	err := s.db.Conn().WithContext(ctx).
		Preload("Category").
		Where("id = ?", idStr).First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
	return &f, err
}

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

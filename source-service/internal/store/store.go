// Package store contains source-service repositories.
// Depends only on ports.DB — no direct postgres imports.
package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/source-service/internal/models"
)

// ErrNotFound is returned when a lookup finds no matching row.
var ErrNotFound = errors.New("not found")

type Store struct{ db ports.DB }

func New(db ports.DB) *Store { return &Store{db: db} }

// CreateArchiveFile persists a newly archived file record.
func (s *Store) CreateArchiveFile(ctx context.Context, f *models.ArchiveFile) error {
	return s.db.Conn().WithContext(ctx).Create(f).Error
}

// GetArchiveFile fetches an archived file by its UUID.
func (s *Store) GetArchiveFile(ctx context.Context, tenantID string, id uuid.UUID) (*models.ArchiveFile, error) {
	var f models.ArchiveFile
	if err := s.db.Conn().WithContext(ctx).First(&f, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

// UpsertBotFileCache stores (or refreshes) the file_id a given bot has cached
// for an archived file.
func (s *Store) UpsertBotFileCache(ctx context.Context, c *models.BotFileCache) error {
	return s.db.Conn().WithContext(ctx).Save(c).Error
}

// GetBotFileCache looks up a cached file_id for a bot, if one exists.
func (s *Store) GetBotFileCache(ctx context.Context, tenantID string, archiveFileID uuid.UUID, botTokenHash string) (*models.BotFileCache, error) {
	var c models.BotFileCache
	err := s.db.Conn().WithContext(ctx).
		Where("tenant_id = ? AND archive_file_id = ? AND bot_token_hash = ?", tenantID, archiveFileID, botTokenHash).
		First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetTelegramSession returns the raw (encrypted) MTProto session bytes
// stored for a phone key, or ErrNotFound if none exists yet.
func (s *Store) GetTelegramSession(ctx context.Context, phone string) ([]byte, error) {
	var row models.TelegramSession
	err := s.db.Conn().WithContext(ctx).Where("phone = ?", phone).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return row.Encrypted, nil
}

// UpsertTelegramSession stores (overwriting) the encrypted session bytes for
// a phone key.
func (s *Store) UpsertTelegramSession(ctx context.Context, phone string, encrypted []byte) error {
	return s.db.Conn().WithContext(ctx).
		Where("phone = ?", phone).
		Assign(models.TelegramSession{Encrypted: encrypted}).
		FirstOrCreate(&models.TelegramSession{Phone: phone, Encrypted: encrypted}).Error
}

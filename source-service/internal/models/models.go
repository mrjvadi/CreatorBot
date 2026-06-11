package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (b *Base) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil { b.ID = uuid.New() }
	return nil
}

type ArchiveFile struct {
	Base
	MessageID int    `gorm:"not null;uniqueIndex"`
	FileType  string `gorm:"not null"`
	FileName  string
	MimeType  string
	FileSize  int64
	Caption   string
}

type BotFileCache struct {
	ArchiveFileID uuid.UUID `gorm:"primaryKey;type:uuid"`
	BotTokenHash  string    `gorm:"primaryKey"` // SHA-256 of token
	FileID        string    `gorm:"not null"`
	CachedAt      time.Time
}

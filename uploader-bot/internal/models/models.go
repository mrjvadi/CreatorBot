package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base is duplicated per-service so each service owns its own DB schema independently.
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (b *Base) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// User is a bot user (Telegram).
type User struct {
	Base
	TelegramID int64  `gorm:"uniqueIndex;not null"`
	Username   string
	FirstName  string
	IsBlocked  bool `gorm:"default:false"`
}

// CodeType defines how many times a code can be used.
type CodeType string

const (
	CodeOnce      CodeType = "once"      // single use
	CodeLimited   CodeType = "limited"   // N uses
	CodeUnlimited CodeType = "unlimited" // infinite
	CodeExpiry    CodeType = "expiry"    // time-limited
)

// Code is a shareable key that delivers one or more files to the user.
type Code struct {
	Base
	Code      string     `gorm:"uniqueIndex;not null"`
	Type      CodeType   `gorm:"not null"`
	MaxUse    int        `gorm:"default:1"`
	UsedCount int        `gorm:"default:0"`
	ExpiresAt *time.Time
	IsAlbum   bool       `gorm:"default:false"` // true = sends all linked files as album
	Files     []CodeFile `gorm:"foreignKey:CodeID"`
}

// File stores a single uploaded Telegram file.
type File struct {
	Base
	UploaderID int64  `gorm:"not null;index"`
	FileID     string `gorm:"not null"` // Telegram file_id
	FileType   string `gorm:"not null"` // document | video | audio | photo | voice | animation | video_note | sticker
	Caption    string
	SourceUUID string // optional UUID from source-service
}

// CodeFile links a Code to one or more Files (ordered).
type CodeFile struct {
	CodeID uuid.UUID `gorm:"primaryKey;type:uuid;index"`
	FileID uuid.UUID `gorm:"primaryKey;type:uuid;index"`
	Order  int       `gorm:"default:0"`
}

// Setting is a key-value store for bot-wide text/config settings.
// Bot owner can change messages like "welcome_text", "not_member_text", etc.
type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

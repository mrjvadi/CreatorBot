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

type User struct {
	Base
	TelegramID int64  `gorm:"uniqueIndex;not null"`
	Username   string
	FirstName  string
	IsBlocked  bool `gorm:"default:false"`
}

type Category struct {
	Base
	Name  string `gorm:"not null;uniqueIndex"`
	Files []File
}

// File is a single archived item with fuzzy-searchable metadata.
// pg_trgm GIN index is created manually in db.Migrate() for the combined column.
type File struct {
	Base
	FileID      string     `gorm:"not null"`     // Telegram file_id
	FileType    string     `gorm:"not null"`     // document | video | audio | photo ...
	Title       string     `gorm:"not null"`
	Tags        string     // comma-separated, e.g. "golang,backend,tutorial"
	Description string
	CategoryID  *uuid.UUID `gorm:"type:uuid;index"`
	Category    *Category  `gorm:"foreignKey:CategoryID"`
	UploaderID  int64
}

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

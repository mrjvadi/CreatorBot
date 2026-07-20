package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
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

// ArchiveFile is a registered file, either archived from a channel message
// (MessageID set to that message's ID) or fetched from a bot via
// run_bot_command (MessageID set to the bot's reply message ID). MessageID
// is intentionally NOT globally unique: message IDs are only unique per
// chat in Telegram, and archived files can now come from many different
// chats/bots, so a single global unique index would eventually collide.
type ArchiveFile struct {
	Base
	TenantID  string `gorm:"index"`
	MessageID int    `gorm:"not null"`
	FileType  string `gorm:"not null"`
	FileName  string
	MimeType  string
	FileSize  int64
	Caption   string
}

type BotFileCache struct {
	TenantID      string    `gorm:"index"`
	ArchiveFileID uuid.UUID `gorm:"primaryKey;type:uuid"`
	BotTokenHash  string    `gorm:"primaryKey"` // SHA-256 of token
	FileID        string    `gorm:"not null"`
	CachedAt      time.Time
}

// TelegramSession holds one account's encrypted MTProto session, keyed by
// phone number, so a lost Docker volume doesn't force a fresh Telegram
// login. Encrypted is opaque ciphertext (AES-256-GCM) — see
// internal/telegram.DBSessionStorage, which is the only code that ever
// decrypts it.
type TelegramSession struct {
	Base
	Phone     string `gorm:"uniqueIndex;not null"` // digits-only phone number
	Encrypted []byte `gorm:"type:bytea;not null"`
}

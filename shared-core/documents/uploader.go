package documents

import (
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BotUser کاربر ربات — داده عملیاتی (نه مهم).
type BotUser struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	TelegramID int64  `bson:"telegram_id"`
	Username   string `bson:"username"`
	FirstName  string `bson:"first_name"`
	IsBlocked  bool   `bson:"is_blocked"`
}

// CodeType نوع کد.
type CodeType string

const (
	CodeOnce      CodeType = "once"
	CodeLimited   CodeType = "limited"
	CodeUnlimited CodeType = "unlimited"
	CodeExpiry    CodeType = "expiry"
)

// Code کد دریافت فایل.
type Code struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	Code      string     `bson:"code"`
	Type      CodeType   `bson:"type"`
	MaxUse    int        `bson:"max_use"`
	UsedCount int        `bson:"used_count"`
	ExpiresAt *time.Time `bson:"expires_at,omitempty"`
	IsAlbum   bool       `bson:"is_album"`
	FileIDs   []string   `bson:"file_ids"` // ObjectID های File
}

// File فایل آپلود شده.
type File struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	TelegramFileID string `bson:"telegram_file_id"`
	FileType       string `bson:"file_type"` // document, video, audio, photo, ...
	Caption        string `bson:"caption"`
	UploaderID     int64  `bson:"uploader_id"`
	SourceUUID     string `bson:"source_uuid,omitempty"`
}

// CodeUsage ثبت استفاده از یک کد (برای آمار).
type CodeUsage struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	CodeID string    `bson:"code_id"`
	UserID int64     `bson:"user_id"`
	UsedAt time.Time `bson:"used_at"`
}

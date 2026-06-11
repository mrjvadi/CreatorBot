package documents

import (
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ArchiveFile فایل آرشیو — با index برای جستجوی فازی.
type ArchiveFile struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	TelegramFileID string   `bson:"telegram_file_id"`
	FileType       string   `bson:"file_type"`
	Title          string   `bson:"title"`
	Tags           []string `bson:"tags"`
	Description    string   `bson:"description"`
	CategoryID     string   `bson:"category_id,omitempty"`
	UploaderID     int64    `bson:"uploader_id"`
	// SearchText فیلد ترکیبی برای text index MongoDB
	SearchText     string   `bson:"search_text"`
}

// ArchiveCategory دسته‌بندی فایل‌های آرشیو.
type ArchiveCategory struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	Name       string `bson:"name"`
	ParentID   string `bson:"parent_id,omitempty"`
}

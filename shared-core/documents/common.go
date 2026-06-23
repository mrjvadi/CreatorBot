package documents

import (
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BotSetting تنظیمات متنی ربات (welcome_text, not_member_text, ...).
type BotSetting struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	Key   string `bson:"key"`
	Value string `bson:"value"`
}

// BotLog لاگ عملیات ربات.
type BotLog struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	Level    string         `bson:"level"` // info, warn, error
	Message  string         `bson:"message"`
	Fields   map[string]any `bson:"fields,omitempty"`
	LoggedAt time.Time      `bson:"logged_at"`
}

// DailyStat آمار روزانه ربات.
type DailyStat struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	Date         string `bson:"date"` // YYYY-MM-DD
	UniqueUsers  int64  `bson:"unique_users"`
	TotalActions int64  `bson:"total_actions"`
	NewUsers     int64  `bson:"new_users"`
}

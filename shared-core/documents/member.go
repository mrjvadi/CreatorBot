package documents

import (
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MemberVerification نتیجه چک عضویت.
type MemberVerification struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	LockID     string    `bson:"lock_id"`   // UUID از PostgreSQL
	UserID     int64     `bson:"user_id"`
	ChannelID  int64     `bson:"channel_id"`
	CheckedBy  string    `bson:"checked_by"` // bot_id
	IsMember   bool      `bson:"is_member"`
	CheckedAt  time.Time `bson:"checked_at"`
}

// MemberLockStat آمار یک lock.
type MemberLockStat struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	LockID       string    `bson:"lock_id"`
	TotalChecks  int64     `bson:"total_checks"`
	PassedChecks int64     `bson:"passed_checks"`
	LastCheck    time.Time `bson:"last_check"`
}

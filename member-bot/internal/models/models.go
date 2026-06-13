package models

import (
	"fmt"
	"strconv"
	"strings"
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

type Owner struct {
	Base
	TelegramID int64   `gorm:"uniqueIndex;not null"`
	Username   string
	FirstName  string
	WalletAddr string
	Balance    float64 `gorm:"default:0"`
	IsBlocked  bool    `gorm:"default:false"`
}

type LockStatus string
const (
	LockActive  LockStatus = "active"
	LockExpired LockStatus = "expired"
)

type Lock struct {
	Base
	OwnerID      uuid.UUID  `gorm:"not null;index"`
	ChannelID    int64      `gorm:"not null;uniqueIndex"`
	ChannelTitle string
	MaxMembers   int        `gorm:"default:0"`
	CurrentCount int        `gorm:"default:0"`
	DurationDay  int        `gorm:"not null"`
	PricePerDay  float64    `gorm:"not null"`
	Status       LockStatus `gorm:"default:'active'"`
	ExpiresAt    time.Time
}

// CheckBot is a sub-bot for getChatMember. Token is AES-256-GCM encrypted.
type CheckBot struct {
	Base
	Token       string                 `gorm:"not null"`
	Username    string
	IsActive    bool                   `gorm:"default:true"`
	RateLimit   int                    `gorm:"default:20"`
	Memberships []BotChannelMembership `gorm:"foreignKey:BotID"`
}

// BotChannelMembership tracks which bots have joined which lock channels.
// Synced to Redis by the dispatcher so workers can filter eligible jobs fast.
type BotChannelMembership struct {
	BotID        uuid.UUID `gorm:"primaryKey;type:uuid;index"`
	ChannelID    int64     `gorm:"primaryKey;index"`
	JoinedAt     time.Time
	LastVerified time.Time
}

type MemberVerification struct {
	Base
	LockID    uuid.UUID `gorm:"not null;index"`
	UserID    int64     `gorm:"not null;index"`
	CheckedBy uuid.UUID `gorm:"type:uuid"`
	IsMember  bool
	CheckedAt time.Time
}

type Payment struct {
	Base
	OwnerID uuid.UUID `gorm:"not null;index"`
	LockID  uuid.UUID `gorm:"not null;index"`
	Amount  float64
	TxHash  string
	Status  string `gorm:"default:'pending'"`
}

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

// BotIDFromToken استخراج Bot ID از توکن تلگرام.
// فرمت توکن: <bot_id>:<random_string>
func BotIDFromToken(token string) (int64, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid token format")
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid bot id: %w", err)
	}
	return id, nil
}

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
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type User struct {
	Base
	TelegramID int64      `gorm:"uniqueIndex;not null"`
	Username   string
	FirstName  string
	Balance    float64    `gorm:"default:0"`
	IsBlocked  bool       `gorm:"default:false"`
	ResellerID *uuid.UUID `gorm:"type:uuid;index"`
	Discount   float64    `gorm:"default:0"` // reseller discount %
}

// Panel stores connection info for a VPN panel instance.
// Type must match a registered ports.VPNPanel adapter name.
type Panel struct {
	Base
	Name       string `gorm:"not null"`
	Type       string `gorm:"not null"` // marzban | marzneshin | hiddify | xui | ...
	BaseURL    string `gorm:"not null"`
	Username   string
	Password   string // AES-256-GCM encrypted
	Capacity   int    `gorm:"default:0"` // 0 = unlimited
	ActiveCount int   `gorm:"default:0"`
	IsActive   bool   `gorm:"default:true"`
}

type Plan struct {
	Base
	Name        string  `gorm:"not null"`
	DurationDay int     `gorm:"not null"`
	DataGB      float64 `gorm:"default:0"` // 0 = unlimited
	Price       float64 `gorm:"not null"`
	IsActive    bool    `gorm:"default:true"`
}

type SubscriptionStatus string

const (
	SubActive   SubscriptionStatus = "active"
	SubExpired  SubscriptionStatus = "expired"
	SubDisabled SubscriptionStatus = "disabled"
)

type Subscription struct {
	Base
	UserID    uuid.UUID          `gorm:"not null;index"`
	User      User               `gorm:"foreignKey:UserID"`
	PanelID   uuid.UUID          `gorm:"not null;index"`
	PlanID    uuid.UUID          `gorm:"not null"`
	Username  string             // panel-side username
	Status    SubscriptionStatus `gorm:"default:'active'"`
	ExpiresAt time.Time
	DataLimit float64
	UsedData  float64
}

type DiscountCode struct {
	Base
	Code      string  `gorm:"uniqueIndex;not null"`
	Percent   float64 `gorm:"not null"`
	MaxUse    int     `gorm:"default:1"`
	UsedCount int     `gorm:"default:0"`
	IsActive  bool    `gorm:"default:true"`
}

type Payment struct {
	Base
	UserID  uuid.UUID `gorm:"not null;index"`
	Amount  float64
	Gateway string // "zarinpal" | "nowpayments" | "card"
	Status  string `gorm:"default:'pending'"`
	RefCode string
	Receipt string // photo file_id for card
	PlanID  *uuid.UUID
}

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

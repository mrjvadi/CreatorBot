// Package models مدل‌های دامنه‌ی vpn-bot را تعریف می‌کند.
//
// این‌ها value object های خالص Go هستند (بدون tag مخصوص هیچ دیتابیسی) — تبدیل
// به/از سند MongoDB درون internal/store انجام می‌شود تا این پکیج به هیچ
// درایوری وابسته نباشد. شکل و نام فیلدها عمداً با نسخه‌ی قبلیِ Postgres یکی
// نگه داشته شده تا کد لایه‌ی handler (tgbot/scheduler) بدون تغییر کار کند.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User کاربر ربات (خریدار اشتراک VPN).
type User struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
	TelegramID int64
	Username   string
	FirstName  string
	Balance    float64
	IsBlocked  bool
	ResellerID *uuid.UUID
	Discount   float64 // reseller discount % — تعریف‌شده، فعلاً بلااستفاده (مثل قبل)
}

// Panel یک پنل VPN (Marzban/X-UI/...).
type Panel struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	Type        string // marzban | marzneshin | hiddify | xui | ...
	BaseURL     string
	Username    string
	Password    string // AES-256-GCM encrypted
	Capacity    int    // 0 = unlimited
	ActiveCount int
	IsActive    bool
}

// Plan پلن فروش اشتراک.
type Plan struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	DurationDay int
	DataGB      float64 // 0 = unlimited
	Price       float64
	IsActive    bool
}

type SubscriptionStatus string

const (
	SubActive   SubscriptionStatus = "active"
	SubExpired  SubscriptionStatus = "expired"
	SubDisabled SubscriptionStatus = "disabled"
)

// Subscription یک اشتراک فعال/تاریخ‌گذشته‌ی کاربر.
type Subscription struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uuid.UUID
	PanelID   uuid.UUID
	PlanID    uuid.UUID
	Username  string // panel-side username
	Status    SubscriptionStatus
	ExpiresAt time.Time
	DataLimit float64
	UsedData  float64
}

// DiscountCode کد تخفیف — تعریف‌شده، فعلاً در مسیر خرید استفاده نمی‌شود (مثل قبل).
type DiscountCode struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	Code      string
	Percent   float64
	MaxUse    int
	UsedCount int
	IsActive  bool
}

// Payment یک پرداخت (آنلاین یا کارت‌به‌کارت).
type Payment struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uuid.UUID
	Amount    float64
	Gateway   string // "zarinpal" | "nowpayments" | "card"
	Status    string // "pending" | "confirmed"
	RefCode   string
	Receipt   string // photo file_id for card
	PlanID    *uuid.UUID
}

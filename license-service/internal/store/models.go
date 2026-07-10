// Package store مدل‌ها و دسترسی DB سرویس لایسنس.
package store

import (
	"time"

	"github.com/google/uuid"
)

// License یک رکورد لایسنس برای یک BotInstance (=instance_id) است.
//
// چرا این سرویس لازم است: هر ربات ساخته‌شده (uploader/vpn/archive/member)
// یک instance_id دارد (=BotID، از توکن استخراج می‌شود). این خودش «کد
// اقتصادی» است — اگر یک نسخه از image یک ربات، همراه با DB connection
// string ها، عیناً از سرور پلتفرم کپی و روی یک سرور دیگر (خارج از کنترل
// agentmanager) اجرا شود، آن clone هم می‌تواند به همان instance_id متصل
// شود و از سهمیه/داده‌های همان مشتری استفاده کند بدون اینکه پولی به
// پلتفرم پرداخت شده باشد. license-service این را با «چسباندن» هر
// instance_id به یک ServerID مشخص، و هشدار در صورت check-in از سرور
// غیرمنتظره، تشخیص می‌دهد.
type License struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time
	UpdatedAt time.Time

	BotID      int64  `gorm:"uniqueIndex;not null"` // instance_id عددی
	InstanceID string `gorm:"index"`                // "bot_<BotID>" — برای خوانایی/لاگ
	OwnerID    string `gorm:"index"`                // uuid کاربر مالک در shared-core
	PlanID     string

	// TokenHash هش SHA-256 توکنِ صادرشده — خودِ توکن ذخیره نمی‌شود (فقط در
	// لحظه‌ی صدور یک‌بار به caller داده می‌شود)، تا نشتِ DB به معنیِ داشتنِ
	// توکن‌های معتبر نباشد.
	TokenHash string `gorm:"index"`

	// KnownServerID سروری که این instance رسماً روی آن deploy شده. هر
	// check-in از سرور دیگری، clone-warning ایجاد می‌کند (اما به‌صورت
	// پیش‌فرض لایسنس را باطل نمی‌کند — fail-open تا مشتری واقعی در جابه‌جایی
	// سرور توسط خودِ پلتفرم قطع نشود؛ ابطال واقعی با license.revoke دستی
	// انجام می‌شود).
	KnownServerID string

	Status         string `gorm:"default:'active';index"` // active|revoked|expired
	RevokedReason  string
	ExpiresAt      *time.Time // nil = بدون انقضا (تا ابطال دستی)
	LastCheckinAt  *time.Time
	LastServerSeen string
	CloneFlagCount int `gorm:"default:0"` // چند بار check-in از سرور غیرمنتظره دیده شده
}

// IsActive بررسی می‌کند لایسنس فعال و منقضی‌نشده است.
func (l *License) IsActive() bool {
	if l.Status != "active" {
		return false
	}
	if l.ExpiresAt != nil && time.Now().After(*l.ExpiresAt) {
		return false
	}
	return true
}

// Package models مدل‌های دامنه‌ی member-bot را تعریف می‌کند — value objectهای
// خالص Go بدون وابستگی به هیچ درایوری (تبدیل به/از سند Mongo در internal/store
// انجام می‌شود). شکل/نام فیلدها با نسخه‌ی قبلیِ Postgres یکی نگه داشته شده تا
// کد لایه‌ی tgbot/dispatcher/scheduler بدون تغییر کار کند.
//
// MemberVerification و Setting نسخه‌ی قبلی حذف شدند — با grep کاملِ کدبیس
// تأیید شد که هیچ‌جا خوانده/نوشته نمی‌شدند (جدولی که فقط ساخته می‌شد و همیشه
// خالی می‌ماند)؛ حمل‌کردنِ schema مرده ارزشی ندارد.
package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Owner مالکِ یک یا چند قفلِ اجاره‌ای.
type Owner struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
	TelegramID int64
	Username   string
	FirstName  string
	WalletAddr string
	Balance    float64
	IsBlocked  bool
}

type LockStatus string

const (
	LockActive  LockStatus = "active"
	LockExpired LockStatus = "expired"
)

// Lock یک قفلِ عضویتِ اجاره‌ای روی یک کانال.
type Lock struct {
	ID           uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	OwnerID      uuid.UUID
	ChannelID    int64
	ChannelTitle string
	MaxMembers   int
	CurrentCount int
	DurationDay  int
	PricePerDay  float64
	Status       LockStatus
	ExpiresAt    time.Time
}

// BotChannelMembership یک عضویتِ CheckBot در یک کانال — در Mongo به‌صورت
// آرایه‌ی embedded داخل خودِ سندِ CheckBot نگه‌داری می‌شود (رجوع store.go)
// چون همیشه با هم خوانده می‌شدند و هرگز join معکوس نداشتند.
type BotChannelMembership struct {
	BotID        uuid.UUID
	ChannelID    int64
	JoinedAt     time.Time
	LastVerified time.Time
}

// CheckBot یک ساب‌بات برای getChatMember. Token با AES-256-GCM رمز شده.
type CheckBot struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Token       string
	Username    string
	IsActive    bool
	RateLimit   int
	Memberships []BotChannelMembership
}

// Payment — مسیر پرداختِ محلیِ owner (فعلاً orphan؛ CreatePayment هیچ‌جا از
// tgbot صدا زده نمی‌شود و ApprovePayment هرگز UpdateBalance را صدا نمی‌زند —
// دقیقاً همان وضعیتِ Postgres قبلی، عمداً بدون تغییرِ رفتار منتقل شد).
type Payment struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	OwnerID   uuid.UUID
	LockID    uuid.UUID
	Amount    float64
	TxHash    string
	Status    string
}

// BotIDFromToken استخراج Bot ID از توکن تلگرام. فرمت: <bot_id>:<random_string>
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

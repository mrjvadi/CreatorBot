// Package models مدل‌های دامنه‌ی archive-bot را تعریف می‌کند — value objectهای
// خالص Go بدون وابستگی به هیچ درایوری. شکل/نام فیلدها با نسخه‌ی قبلیِ Postgres
// یکی نگه داشته شده تا کد لایه‌ی tgbot بدون تغییرِ زیاد کار کند.
//
// Setting حذف شد — با grep کاملِ کدبیس تأیید شد هیچ‌جا خوانده/نوشته نمی‌شد.
// File.Category (نسخه‌ی embedded از Preload قبلی) هم حذف شد — فقط CategoryID
// در کل کدِ handler خوانده می‌شود، هرگز خودِ Category تودرتو.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User کاربرِ ربات.
type User struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
	TelegramID int64
	Username   string
	FirstName  string
	IsBlocked  bool
}

// Category دسته‌بندیِ فایل‌ها.
type Category struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
}

// File یک فایلِ آرشیوشده با متادیتای قابل‌جستجوی فازی.
type File struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FileID      string // Telegram file_id
	FileType    string // document | video | audio | photo ...
	Title       string
	Tags        string // comma-separated
	Description string
	CategoryID  *uuid.UUID
	UploaderID  int64
}

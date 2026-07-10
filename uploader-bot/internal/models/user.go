package models

import "time"

// ── User ─────────────────────────────────────────────────────

type User struct {
	Base          `bson:",inline"`
	TelegramID    int64      `bson:"telegram_id"`
	Username      string     `bson:"username"`
	FirstName     string     `bson:"first_name"`
	IsBlocked     bool       `bson:"is_blocked"`
	FreeDownloads int        `bson:"free_downloads"` // تعداد دانلود رایگان مصرف‌شده
	SubExpiresAt  *time.Time `bson:"sub_expires_at,omitempty"`
	SubPlanID     string     `bson:"sub_plan_id"`
}

// HasActiveSub اشتراک فعال دارد یا نه.
func (u *User) HasActiveSub() bool {
	return u.SubExpiresAt != nil && u.SubExpiresAt.After(time.Now())
}

// ── Admin ─────────────────────────────────────────────────────

type Admin struct {
	Base       `bson:",inline"`
	TelegramID int64    `bson:"telegram_id"`
	Username   string   `bson:"username"`
	IsOwner    bool     `bson:"is_owner"`
	Perms      []string `bson:"perms"` // لیست دسترسی‌ها (مالک همه را دارد)
}

// دسترسی‌های ادمین.
const (
	PermUpload    = "upload"    // آپلود رسانه و مدیریت کدها
	PermBroadcast = "broadcast" // ارسال همگانی
	PermUsers     = "users"     // مدیریت کاربران
	PermLocks     = "locks"     // مدیریت قفل‌ها
	PermSettings  = "settings"  // تنظیمات
	PermAdmins    = "admins"    // مدیریت ادمین‌ها
	PermPlans     = "plans"     // اشتراک/پرداخت
	PermStats     = "stats"     // آمار
	PermBackup    = "backup"    // بکاپ/ریستور
)

// AllPerms همه‌ی دسترسی‌های قابل‌تنظیم.
func AllPerms() []string {
	return []string{PermUpload, PermBroadcast, PermUsers, PermLocks, PermSettings, PermAdmins, PermPlans, PermStats, PermBackup}
}

// Has بررسی می‌کند ادمین یک دسترسی را دارد (مالک همیشه دارد).
func (a *Admin) Has(perm string) bool {
	if a.IsOwner {
		return true
	}
	for _, p := range a.Perms {
		if p == perm {
			return true
		}
	}
	return false
}

// ── Reaction ──────────────────────────────────────────────────

// Reaction واکنش واقعی کاربر روی یک کد (لایک/دیسلایک).
type Reaction struct {
	Base   `bson:",inline"`
	Code   string `bson:"code"`
	UserID int64  `bson:"user_id"`
	Value  int    `bson:"value"` // 1 = لایک، -1 = دیسلایک
}

// ── Download Log ──────────────────────────────────────────────

// DownloadLog شمارش دانلود هر کاربر برای هر کد.
type DownloadLog struct {
	Base   `bson:",inline"`
	UserID string `bson:"user_id"`
	CodeID string `bson:"code_id"`
	Count  int    `bson:"count"`
}

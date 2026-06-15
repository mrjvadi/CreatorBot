package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
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

// ── User ─────────────────────────────────────────────────────

type User struct {
	Base
	TelegramID    int64 `gorm:"uniqueIndex;not null"`
	Username      string
	FirstName     string
	IsBlocked     bool       `gorm:"default:false"`
	FreeDownloads int        `gorm:"default:0"` // تعداد دانلود رایگان استفاده‌شده
	SubExpiresAt  *time.Time // انقضای اشتراک
	SubPlanID     *uuid.UUID `gorm:"index"`
}

func (u *User) HasActiveSub() bool {
	return u.SubExpiresAt != nil && u.SubExpiresAt.After(time.Now())
}

// ── Subscription Plan ─────────────────────────────────────────

type SubPlan struct {
	Base
	Name      string  `gorm:"not null"`
	Price     float64 `gorm:"not null"` // قیمت به تومان یا TON
	Days      int     `gorm:"not null"` // مدت به روز
	IsActive  bool    `gorm:"default:true"`
	SortOrder int     `gorm:"default:0"`
}

// ── Payment ───────────────────────────────────────────────────

type PaymentStatus string
type PaymentGateway string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentConfirmed PaymentStatus = "confirmed"
	PaymentFailed    PaymentStatus = "failed"

	GatewayZarinpal PaymentGateway = "zarinpal"
	GatewayZibal    PaymentGateway = "zibal"
	GatewayCard     PaymentGateway = "card"
	GatewayTON      PaymentGateway = "ton"
	GatewayTRON     PaymentGateway = "tron"
	GatewayStars    PaymentGateway = "stars"
)

type Payment struct {
	Base
	UserID      uuid.UUID      `gorm:"not null;index"`
	PlanID      uuid.UUID      `gorm:"not null;index"`
	Gateway     PaymentGateway `gorm:"not null"`
	Amount      float64        `gorm:"not null"`
	Status      PaymentStatus  `gorm:"default:'pending';index"`
	Authority   string         // zarinpal/zibal authority code
	TxHash      string         // crypto tx hash
	CardRef     string         // card-to-card photo file_id
	Stars       int            // telegram stars amount
	ConfirmedAt *time.Time
}

// ── Folder / Category ─────────────────────────────────────────

type Folder struct {
	Base
	Name      string     `gorm:"not null"`
	ParentID  *uuid.UUID `gorm:"index"` // nil = root folder
	Icon      string     // emoji
	SortOrder int        `gorm:"default:0"`
	IsActive  bool       `gorm:"default:true"`
	Children  []Folder   `gorm:"foreignKey:ParentID"`
}

// ── Media Code ────────────────────────────────────────────────

type CodeType string

const (
	CodeOnce      CodeType = "once"
	CodeLimited   CodeType = "limited"
	CodeUnlimited CodeType = "unlimited"
	CodeExpiry    CodeType = "expiry"
)

type Code struct {
	Base
	Code     string     `gorm:"uniqueIndex;not null"` // کد رسانه (قابل تغییر)
	Type     CodeType   `gorm:"not null"`
	FolderID *uuid.UUID `gorm:"index"` // پوشه

	// تنظیمات محتوا
	Caption   string
	Thumbnail string // file_id تامبنیل
	IsAlbum   bool   `gorm:"default:false"`

	// محدودیت‌ها
	MaxUse    int `gorm:"default:1"`
	UsedCount int `gorm:"default:0"`
	ExpiresAt *time.Time

	// قفل‌ها
	ForwardLock   bool   `gorm:"default:false"` // قفل فوروارد
	ChannelLock   bool   `gorm:"default:false"` // جوین اجباری
	AutoDelete    int    `gorm:"default:0"`     // ثانیه (0=غیرفعال)
	Password      string // رمز عبور (خالی=بدون رمز)
	DownloadLimit int    `gorm:"default:0"` // محدودیت دانلود (0=نامحدود)

	// fake stats
	FakeLikes     int `gorm:"default:0"`
	FakeDownloads int `gorm:"default:0"`
	FakeViews     int `gorm:"default:0"`

	// اشتراک
	SubRequired bool `gorm:"default:false"` // نیاز به اشتراک

	UploaderID int64 `gorm:"index"` // telegram_id آپلود کننده

	Files []CodeFile `gorm:"foreignKey:CodeID"`
}

// File یک فایل تلگرام.
type File struct {
	Base
	FileID     string `gorm:"not null"`
	FileType   string `gorm:"not null"` // video|photo|audio|document|animation|voice|sticker
	Caption    string
	Thumbnail  string // file_id کاور (برای ویدیو)
	SourceUUID string
}

// CodeFile رابطه Code و File.
type CodeFile struct {
	CodeID uuid.UUID `gorm:"primaryKey;type:uuid;index"`
	FileID uuid.UUID `gorm:"primaryKey;type:uuid;index"`
	Order  int       `gorm:"default:0"`
}

// ── Force Join Channel ────────────────────────────────────────

type ForceJoinChannel struct {
	Base
	ChatID    int64 `gorm:"uniqueIndex;not null"`
	Title     string
	Username  string // برای t.me/username
	InviteURL string // برای private channel
	CheckBot  bool   `gorm:"default:false"` // ربات واسط
	IsActive  bool   `gorm:"default:true"`
	SortOrder int    `gorm:"default:0"`
}

// ── Preview Channel ───────────────────────────────────────────

type PreviewChannel struct {
	Base
	ChatID   int64 `gorm:"uniqueIndex;not null"`
	Title    string
	IsActive bool `gorm:"default:true"`
}

// ── Backup ────────────────────────────────────────────────────

type Backup struct {
	Base
	FileID     string `gorm:"not null"` // telegram file_id فایل بکاپ
	FileSize   int64
	TotalCodes int
	TotalFiles int
	CreatedBy  int64 // telegram_id ادمین
}

// ── Setting ───────────────────────────────────────────────────

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

// تنظیمات پیش‌فرض
const (
	SettingWelcomeText        = "welcome_text"
	SettingNotMemberText      = "not_member_text"
	SettingPasswordText       = "password_text"
	SettingSubRequiredText    = "sub_required_text"
	SettingBotActive          = "bot_active"
	SettingFreeDownloads      = "free_downloads"      // تعداد دانلود رایگان
	SettingAutoDeleteDefault  = "auto_delete_default" // ثانیه
	SettingForwardLockDefault = "forward_lock_default"
	SettingSubRequired        = "sub_required" // اشتراک اجباری global
	SettingUserUpload         = "user_upload"  // آپلود توسط کاربر
	SettingAutoApproveFiles   = "auto_approve_files"
	SettingShowSearch         = "show_search"
	SettingShowLikesButtons   = "show_likes"
	SettingShowReportButton   = "show_report"
	SettingSignature          = "signature"  // امضای زیر فایل‌ها
	SettingSpamDelay          = "spam_delay" // ثانیه
	SettingPaymentZarinpal    = "payment_zarinpal"
	SettingPaymentZibal       = "payment_zibal"
	SettingPaymentCard        = "payment_card"
	SettingPaymentTON         = "payment_ton"
	SettingPaymentTRON        = "payment_tron"
	SettingPaymentStars       = "payment_stars"
	SettingActiveGateway      = "active_gateway" // zarinpal|zibal
	SettingZarinpalMerchant   = "zarinpal_merchant"
	SettingZibalMerchant      = "zibal_merchant"
	SettingCardNumber         = "card_number"
	SettingCardHolder         = "card_holder"
	SettingTONWallet          = "ton_wallet"
	SettingTRONWallet         = "tron_wallet"
	SettingBroadcastInterval  = "broadcast_interval" // دقیقه
	SettingVideoThumbDefault  = "video_thumb_default"
)

// ── User Download History ─────────────────────────────────────

type DownloadLog struct {
	Base
	UserID uuid.UUID `gorm:"not null;index"`
	CodeID uuid.UUID `gorm:"not null;index"`
	Count  int       `gorm:"default:1"`
}

// ── Admin ─────────────────────────────────────────────────────

type Admin struct {
	Base
	TelegramID int64 `gorm:"uniqueIndex;not null"`
	Username   string
	IsOwner    bool `gorm:"default:false"`
}

// AllModels برای AutoMigrate.
func AllModels() []any {
	return []any{
		&User{}, &SubPlan{}, &Payment{},
		&Folder{}, &Code{}, &File{}, &CodeFile{},
		&ForceJoinChannel{}, &PreviewChannel{},
		&Backup{}, &Setting{}, &DownloadLog{}, &Admin{},
	}
}

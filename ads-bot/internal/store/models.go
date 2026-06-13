// Package store مدل‌های سیستم تبلیغات هوشمند.
//
// معماری:
//   - ادمین اصلی CPJ پایه را تعیین می‌کند (نه publisher)
//   - هر کانال امتیاز (Score) دارد که از تحلیل ممبرها محاسبه می‌شود
//   - کانال‌ها دسته‌بندی دارند (ارز دیجیتال، سینما، تکنولوژی، ...)
//   - هر ممبر درصد fake بودن دارد → CPJ واقعی بر اساس ممبرهای real
package store

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── تنظیمات ادمین ─────────────────────────────────────────

// AdConfig تنظیمات سیستم تبلیغات که ادمین اصلی مشخص می‌کند.
type AdConfig struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UpdatedAt time.Time

	// هزینه پایه به ازای هر عضو واقعی (TON)
	BaseCPJ float64 `gorm:"not null;default:0.005"`

	// ضریب دسته‌بندی روی CPJ اعمال می‌شود
	// مثلاً کانال‌های crypto ضریب ۲ دارند

	// حداقل امتیاز کانال برای پذیرش تبلیغ (0-100)
	MinChannelScore int `gorm:"default:30"`

	// حداکثر درصد فیک مجاز (اگه بیشتر بود CPJ کاهش می‌یابد)
	MaxFakePercent float64 `gorm:"default:30.0"`

	// کمیسیون پلتفرم از هر تبلیغ (درصد)
	PlatformCommission float64 `gorm:"default:20.0"`

	IsActive bool `gorm:"default:true"`
}

// ── دسته‌بندی کانال ───────────────────────────────────────

// ChannelCategory دسته‌بندی کانال‌ها.
type ChannelCategory struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt   time.Time
	Name        string  `gorm:"uniqueIndex;not null"` // crypto | cinema | tech | sport | ...
	Label       string  // نام فارسی: ارز دیجیتال، سینما، ...
	CPJMultiplier float64 `gorm:"default:1.0"` // ضریب CPJ برای این دسته
	IsActive    bool    `gorm:"default:true"`
}

// ── Publisher ──────────────────────────────────────────────

type Publisher struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	TelegramID int64     `gorm:"uniqueIndex;not null"`
	Username   string
	FirstName  string
	Balance    float64 `gorm:"default:0"`
	IsBlocked  bool    `gorm:"default:false"`
}

// ── کانال ────────────────────────────────────────────────

// ChannelStatus وضعیت کانال.
type ChannelStatus string

const (
	ChannelPending  ChannelStatus = "pending"
	ChannelVerified ChannelStatus = "verified"
	ChannelRejected ChannelStatus = "rejected"
	ChannelSuspended ChannelStatus = "suspended"
)

// AdChannel کانالی که در شبکه تبلیغات ثبت شده.
type AdChannel struct {
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	OwnerID     uuid.UUID     `gorm:"not null;index"`
	CategoryID  *uuid.UUID    `gorm:"type:uuid;index"`
	Category    *ChannelCategory `gorm:"foreignKey:CategoryID"`

	// اطلاعات کانال
	ChannelID   int64         `gorm:"uniqueIndex;not null"`
	ChannelName string
	ChannelUsername string
	MemberCount int           `gorm:"default:0"`

	// وضعیت
	Status      ChannelStatus `gorm:"default:'pending';index"`
	IsActive    bool          `gorm:"default:true"`

	// امتیاز کانال (0-100) — از تحلیل ممبرها
	Score       int           `gorm:"default:0"`
	// درصد احتمال fake بودن ممبرها (0-100)
	FakePercent float64       `gorm:"default:0"`
	// تعداد ممبر real (بر اساس تحلیل)
	RealMembers int           `gorm:"default:0"`

	// CPJ موثر = BaseCPJ × category.CPJMultiplier × (1 - fake_ratio)
	EffectiveCPJ float64      `gorm:"default:0"`

	// آخرین تحلیل
	LastAnalyzedAt *time.Time

	// آمار
	TotalImpressions int  `gorm:"default:0"`
	TotalEarned      float64 `gorm:"default:0"`
}

// ── تحلیل ممبر ────────────────────────────────────────────

// MemberAnalysis نتیجه تحلیل یک ممبر.
type MemberAnalysis struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt   time.Time

	ChannelID   int64     `gorm:"not null;index"`
	TelegramID  int64     `gorm:"not null;index"`

	// فاکتورهای تشخیص fake
	HasUsername     bool    `gorm:"default:false"`
	HasProfilePhoto bool    `gorm:"default:false"`
	AccountAge      int     // تخمین سن اکانت به روز (0 = نامشخص)
	IsBot           bool    `gorm:"default:false"`

	// امتیاز واقعی بودن (0-100)
	RealScore   int     `gorm:"default:0"`
	IsFake      bool    `gorm:"default:false"`

	AnalyzedAt  time.Time
}

// ── کمپین ────────────────────────────────────────────────

type CampaignStatus string

const (
	CampaignDraft    CampaignStatus = "draft"
	CampaignPending  CampaignStatus = "pending"
	CampaignActive   CampaignStatus = "active"
	CampaignPaused   CampaignStatus = "paused"
	CampaignDone     CampaignStatus = "done"
	CampaignRejected CampaignStatus = "rejected"
)

type Campaign struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	PublisherID uuid.UUID      `gorm:"not null;index"`
	Name        string         `gorm:"not null"`
	Status      CampaignStatus `gorm:"default:'draft';index"`

	// فیلتر دسته‌بندی هدف (اختیاری)
	TargetCategoryID *uuid.UUID `gorm:"type:uuid"`
	TargetCategory   *ChannelCategory `gorm:"foreignKey:TargetCategoryID"`

	// حداقل امتیاز کانال هدف (اختیاری - پیش‌فرض از AdConfig)
	MinChannelScore *int

	// محتوا
	MediaFileID string
	MediaType   string
	Caption     string `gorm:"type:text"`
	ButtonText  string
	ButtonURL   string

	// بودجه
	Budget float64 `gorm:"not null"`
	Spent  float64 `gorm:"default:0"`

	// هزینه به ازای هر عضو واقعی (از AdConfig.BaseCPJ × category multiplier)
	CPJ float64 `gorm:"not null"`

	// آمار
	TotalJoins    int     `gorm:"default:0"` // کل join ها
	RealJoins     int     `gorm:"default:0"` // فقط join های real
	TargetCount   int     `gorm:"default:0"`

	// زمان
	StartAt *time.Time
	EndAt   *time.Time

	// تأیید
	ReviewNote string
	ReviewedAt *time.Time
	ReviewerID int64
}

func (c *Campaign) RemainingBudget() float64 { return c.Budget - c.Spent }
func (c *Campaign) IsExpired() bool {
	return c.EndAt != nil && time.Now().After(*c.EndAt)
}

// ── Impression ────────────────────────────────────────────

type Impression struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt  time.Time

	CampaignID uuid.UUID `gorm:"not null;index"`
	ChannelID  int64     `gorm:"not null;index"`
	MessageID  int

	// join های این impression
	TotalJoins int     `gorm:"default:0"`
	RealJoins  int     `gorm:"default:0"`
	FakeJoins  int     `gorm:"default:0"`

	// هزینه پرداخت‌شده به صاحب کانال (فقط برای real join ها)
	Cost float64 `gorm:"default:0"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AdConfig{},
		&ChannelCategory{},
		&Publisher{},
		&AdChannel{},
		&MemberAnalysis{},
		&Campaign{},
		&Impression{},
	)
}

// DefaultCategories دسته‌بندی‌های پیش‌فرض.
func DefaultCategories() []ChannelCategory {
	return []ChannelCategory{
		{Name: "crypto",      Label: "ارز دیجیتال",  CPJMultiplier: 2.0},
		{Name: "tech",        Label: "تکنولوژی",      CPJMultiplier: 1.5},
		{Name: "cinema",      Label: "سینما و فیلم",  CPJMultiplier: 1.2},
		{Name: "sport",       Label: "ورزش",          CPJMultiplier: 1.3},
		{Name: "news",        Label: "خبر و اخبار",   CPJMultiplier: 1.4},
		{Name: "education",   Label: "آموزش",         CPJMultiplier: 1.3},
		{Name: "entertainment",Label: "سرگرمی",       CPJMultiplier: 1.0},
		{Name: "business",    Label: "کسب و کار",     CPJMultiplier: 1.8},
		{Name: "health",      Label: "سلامت",         CPJMultiplier: 1.2},
		{Name: "other",       Label: "سایر",          CPJMultiplier: 1.0},
	}
}

// DefaultConfig تنظیم پیش‌فرض.
func DefaultConfig() AdConfig {
	return AdConfig{
		BaseCPJ:            0.005,
		MinChannelScore:    30,
		MaxFakePercent:     30.0,
		PlatformCommission: 20.0,
		IsActive:           true,
	}
}

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
		// lock-rental tables
		&LockRentalCampaign{},
		&FreeBotSlot{},
		&RentalJoinReward{},
		&FreeBotOwnerReward{},
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

// ══════════════════════════════════════════════════════════════
// Lock Rental — اجاره‌ی قفل کانال روی ربات‌های رایگان پلتفرم
//
// تفاوت با Campaign (CPJ) بالا: در آنجا advertiser پول می‌دهد تا تبلیغش را
// در کانال *دیگران* نشان دهد و صاحب کانال پول می‌گیرد. اینجا برعکس است:
// خریدار (owner کانال خودش) پول می‌دهد تا کانالش به‌عنوان قفل عضویت روی
// ربات‌های رایگان ما قرار بگیرد، و این بار کاربری که عضو کانال او می‌شود
// پاداش می‌گیرد (نه صاحب کانال). هزینه از موجودی خریدار کسر می‌شود.
// ══════════════════════════════════════════════════════════════

// LockRentalStatus وضعیت یک درخواست اجاره.
type LockRentalStatus string

const (
	RentalPendingReview LockRentalStatus = "pending_review" // منتظر تأیید ادمین اصلی
	RentalActive        LockRentalStatus = "active"          // تأیید شده و در حال اجرا
	RentalPaused         LockRentalStatus = "paused"
	RentalRejected       LockRentalStatus = "rejected"
	RentalDone           LockRentalStatus = "done" // بودجه تمام شد یا منقضی شد
)

// LockRentalCampaign یک درخواست اجاره‌ی قفل کانال.
type LockRentalCampaign struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// خریدار — کسی که می‌خواهد کانالش به‌عنوان قفل روی ربات‌های رایگان قرار بگیرد
	BuyerTelegramID int64 `gorm:"not null;index"`

	// کانال هدف که کاربران باید عضوش شوند
	TargetChannelID       int64  `gorm:"not null;index"`
	TargetChannelUsername string

	Status     LockRentalStatus `gorm:"default:'pending_review';index"`
	ReviewNote string
	ReviewedAt *time.Time
	ReviewerID int64 // باید همیشه OWNER_ID پلتفرم باشد، نه ادمین معمولی

	// اقتصاد — هزینه به ازای هر عضو واقعی که از طریق ربات‌های رایگان جذب می‌شود
	RewardPerJoinTON float64 `gorm:"not null"` // پرداختی به کاربر عضوشونده
	Budget           float64 `gorm:"not null"` // کل بودجه‌ای که خریدار کنار گذاشته (از کیف پولش کسر می‌شود)
	Spent            float64 `gorm:"default:0"`

	// FreeBotOwnerRewardPercent درصدی از کل بودجه که بین owner های واقعی
	// ربات‌های رایگان (کسانی که از botmanager این bot ها را ساخته‌اند، نه
	// خریدار اجاره) تقسیم می‌شود — جمعی، نه per-join. مثلا 5 یعنی 5% کل
	// بودجه بین همه‌ی owner های slot های این کمپین تقسیم می‌شود.
	FreeBotOwnerRewardPercent float64 `gorm:"default:5"`

	TotalJoins int `gorm:"default:0"`
	RealJoins  int `gorm:"default:0"`

	StartAt *time.Time
	EndAt   *time.Time
}

func (l *LockRentalCampaign) RemainingBudget() float64 { return l.Budget - l.Spent }
func (l *LockRentalCampaign) IsActive() bool {
	if l.Status != RentalActive {
		return false
	}
	if l.RemainingBudget() <= 0 {
		return false
	}
	if l.EndAt != nil && time.Now().After(*l.EndAt) {
		return false
	}
	return true
}

// FreeBotSlot یک ربات رایگان پلتفرم که می‌تواند به یک LockRentalCampaign
// وصل شود. وقتی RentalID خالی است یعنی این ربات فعلاً قفل تبلیغ خودِ
// پلتفرم را دارد (رایگان)؛ وقتی پر است یعنی به یک کمپین اجاره‌ای وصل است.
//
// نکته: BotInstanceID به shared-core/models.BotInstance.ID اشاره می‌کند
// (ads-bot عمداً مدل botmanager را import نمی‌کند تا coupling نداشته باشد؛
// فقط uuid آن را نگه می‌دارد).
type FreeBotSlot struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time
	UpdatedAt time.Time

	BotInstanceID uuid.UUID `gorm:"uniqueIndex;not null"` // یک instance فقط یک slot دارد
	BotID         int64     `gorm:"uniqueIndex;not null"` // BotID تلگرامی، برای lookup سریع

	// RentalID — کمپین اجاره‌ای که این ربات الان به آن وصل است (nil = آزاد/رایگان پلتفرم)
	RentalID *uuid.UUID `gorm:"index"`

	// AssignedOwnerTelegramID — کسی که این ربات رایگان الان "در اختیارش" است
	// (بعد از تأیید ادمین، ربات‌ها در اختیار خریدار قرار می‌گیرند طبق گفته‌ی کاربر)
	AssignedOwnerTelegramID int64

	// IsChannelAdminConfirmed — وقتی خریدار ربات را در کانال خودش ادمین کرد،
	// این true می‌شود و از همان لحظه سرویس شروع به قفل‌کردن می‌کند.
	IsChannelAdminConfirmed bool `gorm:"default:false"`
}

func (s *FreeBotSlot) IsFree() bool   { return s.RentalID == nil }
func (s *FreeBotSlot) IsRented() bool { return s.RentalID != nil }

// RentalJoinReward ثبت یک پاداش per-join که پرداخت شده — یونیک‌بودن
// (RentalID, TelegramID) تضمین می‌کند هر کاربر برای یک کمپین فقط یک‌بار
// پاداش بگیرد، حتی اگر membership.joined دوبار برسد (at-least-once NATS).
// JoinRewardStatus وضعیت تسویه‌ی یک پاداش per-join.
type JoinRewardStatus string

const (
	// RewardPending — join ثبت و از بودجه‌ی کمپین "رزرو" شده، ولی هنوز
	// واقعاً به کاربر واریز نشده (در دوره‌ی انتظار برای تشخیص تقلب).
	RewardPending JoinRewardStatus = "pending"
	// RewardSettled — بعد از گذشت مهلت، واقعاً واریز شده.
	RewardSettled JoinRewardStatus = "settled"
	// RewardReversed — قبل از تسویه، fraud-engine این join را رد کرد؛
	// هرگز واریز نمی‌شود (ولی بودجه‌ی رزروشده باید به کمپین برگردد).
	RewardReversed JoinRewardStatus = "reversed"
)

// RentalJoinReward ثبت یک پاداش per-join — یونیک‌بودن (RentalID, TelegramID)
// تضمین می‌کند هر کاربر برای یک کمپین فقط یک‌بار پاداش بگیرد، حتی اگر
// membership.joined دوبار برسد (at-least-once NATS).
//
// تسویه با تأخیر است: لحظه‌ی join فقط بودجه "رزرو" می‌شود (Spent در
// LockRentalCampaign بالا می‌رود تا بودجه‌ی باقی‌مانده درست حساب شود)،
// ولی واریز واقعی به کیف پول کاربر تا SettleAt اتفاق نمی‌افتد — فرصتی
// برای تشخیص تقلب قبل از پرداخت نهایی.
// FreeBotOwnerReward ثبت سهم یک owner ربات رایگان از یک کمپین — یونیک‌بودن
// (RentalID, SlotID) تضمین می‌کند هر slot برای یک کمپین فقط یک‌بار سهم
// حساب شود. همان state machine تأخیری RentalJoinReward را دارد (طبق
// انسجام حسابداری: کسر بودجه همان لحظه‌ی تأیید کمپین انجام شده، ولی واریز
// واقعی به owner با همان تأخیر RewardSettlementDelay صورت می‌گیرد).
// RentalJoinReward ثبت یک پاداش per-join — یونیک‌بودن (RentalID, TelegramID)
// تضمین می‌کند هر کاربر برای یک کمپین فقط یک‌بار پاداش بگیرد، حتی اگر
// membership.joined دوبار برسد (at-least-once NATS).
//
// تسویه با تأخیر است: لحظه‌ی join فقط بودجه "رزرو" می‌شود (Spent در
// LockRentalCampaign بالا می‌رود تا بودجه‌ی باقی‌مانده درست حساب شود)،
// ولی واریز واقعی به کیف پول کاربر تا SettleAt اتفاق نمی‌افتد — فرصتی
// برای تشخیص تقلب قبل از پرداخت نهایی.
type RentalJoinReward struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt  time.Time
	RentalID   uuid.UUID `gorm:"uniqueIndex:idx_rental_user"`
	TelegramID int64     `gorm:"uniqueIndex:idx_rental_user"`
	AmountTON  float64

	Status    JoinRewardStatus `gorm:"default:'pending';index"`
	SettleAt  time.Time        `gorm:"index"` // CreatedAt + مهلت انتظار (پیش‌فرض 24h)
	SettledAt *time.Time
}

// FreeBotOwnerReward ثبت سهم یک owner ربات رایگان از یک کمپین — یونیک‌بودن
// (RentalID, SlotID) تضمین می‌کند هر slot برای یک کمپین فقط یک‌بار سهم
// حساب شود. همان state machine تأخیری RentalJoinReward را دارد (طبق
// انسجام حسابداری: کسر بودجه همان لحظه‌ی تأیید کمپین انجام شده، ولی واریز
// واقعی به owner با همان تأخیر RewardSettlementDelay صورت می‌گیرد).
type FreeBotOwnerReward struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt       time.Time
	RentalID        uuid.UUID `gorm:"uniqueIndex:idx_rental_slot"`
	SlotID          uuid.UUID `gorm:"uniqueIndex:idx_rental_slot"`
	OwnerTelegramID int64
	AmountTON       float64

	Status    JoinRewardStatus `gorm:"default:'pending';index"`
	SettleAt  time.Time        `gorm:"index"`
	SettledAt *time.Time
}

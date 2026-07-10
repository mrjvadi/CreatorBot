// Package models — مدل‌های MongoDB برای admanager-bot.
//
// این ربات یک ابزار ادمین‌محور است: صاحب/ادمین کانال‌ها ربات را به
// کانال‌های خودش اضافه می‌کند تا مدیریت تبلیغات، زمان‌بندی و آمار را
// راحت‌تر انجام دهد. هیچ نقش «مشتری» یا لایه‌ی پرداختی در سیستم وجود ندارد.
//
// تمام مدل‌ها با instance_id ایزوله می‌شوند تا داده‌ی هر ربات
// از بقیه جدا بماند. شناسه‌ها از نوع string (UUID) هستند.
package models

import "time"

// مقادیر پیش‌فرض ساعت فعال روزانه‌ی کمپین‌ها (۸ صبح تا ۲۳).
const (
	DefaultStartHour = 8
	DefaultEndHour   = 23

	// DefaultIntervalMinutes فاصله‌ی پیش‌فرض بین پست‌های یک کمپین.
	DefaultIntervalMinutes = 60

	// DefaultReminderMinutesBefore پیش‌فرض یادآوریِ پیش از ارسال (دقیقه).
	DefaultReminderMinutesBefore = 10
)

// ── Channel ─────────────────────────────────────────────────────

// ChannelStatus وضعیت یک کانال.
type ChannelStatus string

const (
	ChannelActive   ChannelStatus = "active"
	ChannelInactive ChannelStatus = "inactive"
	ChannelPending  ChannelStatus = "pending"
)

// Channel یک کانال تلگرامی که ادمین در سیستم ثبت کرده.
type Channel struct {
	ID          string        `bson:"_id"`
	InstanceID  string        `bson:"instance_id"`
	TelegramID  int64         `bson:"telegram_id"`
	Username    string        `bson:"username"`
	Title       string        `bson:"title"`
	MemberCount int           `bson:"member_count"`
	Status      ChannelStatus `bson:"status"`
	TagIDs      []string      `bson:"tag_ids"` // ← Tag.ID
	// آمار
	AvgViews   int     `bson:"avg_views"`
	EngageRate float64 `bson:"engage_rate"`
	// تاریخ‌ها
	CreatedAt time.Time  `bson:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty"`
}

// Tag برچسب دسته‌بندی کانال.
type Tag struct {
	ID         string    `bson:"_id"`
	InstanceID string    `bson:"instance_id"`
	Name       string    `bson:"name"`
	Slug       string    `bson:"slug"`      // برای جستجوی سریع
	ParentID   string    `bson:"parent_id"` // برچسب والد (درخت)
	IsActive   bool      `bson:"is_active"`
	CreatedAt  time.Time `bson:"created_at"`
}

// ── Campaign ─────────────────────────────────────────────────────

// CampaignStatus وضعیت کمپین.
type CampaignStatus string

const (
	CampaignDraft     CampaignStatus = "draft"     // پیش‌نویس
	CampaignRunning   CampaignStatus = "running"   // در حال اجرا
	CampaignPaused    CampaignStatus = "paused"    // توقف موقت
	CampaignCompleted CampaignStatus = "completed" // پایان یافته
	CampaignCancelled CampaignStatus = "cancelled" // لغو شده
)

// Campaign یک کمپین تبلیغاتی که ادمین برای کانال‌های خود تعریف می‌کند.
//
// کمپین صرفاً یک گروه‌بندی از تبلیغ‌ها همراه با زمان‌بندی و هدف‌گذاری
// روی کانال‌هاست؛ هیچ بودجه‌ی مالی یا workflow تأیید ندارد.
type Campaign struct {
	ID         string         `bson:"_id"`
	InstanceID string         `bson:"instance_id"`
	Name       string         `bson:"name"`
	Status     CampaignStatus `bson:"status"`

	// زمان‌بندی روزانه — بازه‌ی «شروع تا پایان». اگر EndHour/EndMinute از
	// StartHour/StartMinute کوچک‌تر باشد، یعنی بازه از نیمه‌شب عبور می‌کند
	// (مثلاً شروع ۲۲:۰۰ و پایان ۰۲:۰۰). وقتی به پایان بازه می‌رسیم، هر
	// چرخه‌ی در حال اجرای این کمپین فوراً قطع می‌شود (نه اینکه بگذاریم
	// طبیعی تمام شود) — بدون اینکه ربات یا سایر کمپین‌ها متوقف شوند.
	StartHour   int `bson:"start_hour"`   // ساعت شروع (0-23)
	StartMinute int `bson:"start_minute"` // دقیقه‌ی شروع (0-59)
	EndHour     int `bson:"end_hour"`     // ساعت پایان روزانه (0-23)
	EndMinute   int `bson:"end_minute"`   // دقیقه‌ی پایان روزانه (0-59)

	IntervalMinutes    int `bson:"interval_minutes"`     // فاصله بین پست‌ها (دقیقه)
	DeleteAfterMinutes int `bson:"delete_after_minutes"` // کل عمر یک چرخه (پست اصلی + ریپلی‌ها)؛ بعد از این مدت همه‌چیز پاک می‌شود (0 = حذف نشود)
	RotationMinutes    int `bson:"rotation_minutes"`     // هر چند دقیقه تبلیغ بعدی از لیست جایگزین شود (0 = بدون چرخش)

	// پایان اختیاری کل کمپین
	EndAt *time.Time `bson:"end_at,omitempty"`

	// هدف‌گذاری
	TargetTagIDs     []string `bson:"target_tag_ids"`     // کانال‌ها با این برچسب‌ها
	TargetChannelIDs []string `bson:"target_channel_ids"` // کانال‌های خاص
	MinMemberCount   int      `bson:"min_member_count"`   // حداقل عضو کانال

	// آمار
	TotalImpressions int `bson:"total_impressions"`
	TotalClicks      int `bson:"total_clicks"`

	CreatedAt time.Time  `bson:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty"`
}

// ── Advertisement ────────────────────────────────────────────────

// Advertisement یک تبلیغ = یک پست اصلی + چند پست ریپلی متوالی.
//
// محتوا به‌صورت ارجاع به پیام‌هایی که ادمین برای ربات فرستاده ذخیره می‌شود
// و هنگام ارسال با Copy بازتولید می‌شود (هر نوع و از هر منبعی، بدون برچسب
// «forwarded from»). همه‌ی پیام‌ها در همان چتِ ادمین با ربات قرار دارند.
//
// چرخه‌ی نمایش: پست اصلی برای کل مدت DeleteAfterMinutes کمپین زنده می‌ماند.
// ریپلی‌ها یکی‌یکی و متوالی نمایش داده می‌شوند — هر AdReply مدت‌زمان
// نمایش خودش را دارد؛ وقتی مدتش تمام شد پاک می‌شود و ریپلی بعدی می‌آید.
// وقتی آخرین ریپلی هم تمام شد (یا اصلاً ریپلی‌ای نبود)، پست اصلی + آخرین
// ریپلیِ در حال نمایش با هم پاک می‌شوند.
type Advertisement struct {
	ID         string `bson:"_id"`
	InstanceID string `bson:"instance_id"`
	CampaignID string `bson:"campaign_id"`
	Name       string `bson:"name"` // نام داخلی برای مدیریت

	SourceChatID  int64 `bson:"source_chat_id"`  // چت ادمین (مبدأ Copy)
	MainMessageID int   `bson:"main_message_id"` // پیام اصلی

	Replies []AdReply `bson:"replies"` // ریپلی‌های متوالی، هرکدام با مدت‌زمان خودش

	// تنظیمات نمایش ثابت/پین — هرکدام مستقل، در سطح همین تبلیغ.
	KeepAsLastMessage      bool `bson:"keep_as_last_message"`      // همیشه آخرین پیام کانال بماند؛ با هر پست جدیدِ دیگر، دوباره فرستاده شود
	PinMessage             bool `bson:"pin_message"`               // با Pin بومی تلگرام پین شود
	DeletePreviousOnRepost bool `bson:"delete_previous_on_repost"` // هنگام بازارسال (به‌خاطر KeepAsLastMessage)، نسخه‌ی قبلی در کانال حذف شود

	IsActive  bool      `bson:"is_active"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// AdReply یک پست ریپلیِ متوالی روی پست اصلیِ یک تبلیغ، با مدت‌زمان
// نمایش مستقل به خودش (به دقیقه) پیش از آنکه پاک شده و ریپلی بعدی بیاید.
type AdReply struct {
	MessageID       int `bson:"message_id"`
	DurationMinutes int `bson:"duration_minutes"`
}

// ── Job (ScheduledJob) ───────────────────────────────────────────

// JobType نوع job زمان‌بندی‌شده.
type JobType string

const (
	JobTypeSendAd      JobType = "send_ad"      // ارسال پست اصلی تبلیغ
	JobTypeSendReply   JobType = "send_reply"   // ارسال یک ریپلی روی پست اصلی (با فاصله)
	JobTypeDeletePost  JobType = "delete_post"  // حذف خودکار پست ارسال‌شده
	JobTypeUpdateStats JobType = "update_stats" // به‌روزرسانی آمار
	JobTypeEndCampaign JobType = "end_campaign" // پایان کمپین
	JobTypeReminder    JobType = "reminder"     // یادآوری
)

// JobStatus وضعیت job.
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobDone      JobStatus = "done"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

// ScheduledJob یک کار برنامه‌ریزی‌شده در صف.
type ScheduledJob struct {
	ID         string    `bson:"_id"`
	InstanceID string    `bson:"instance_id"`
	Type       JobType   `bson:"type"`
	Status     JobStatus `bson:"status"`

	// داده‌ی مرتبط
	CampaignID      string `bson:"campaign_id,omitempty"`
	AdvertisementID string `bson:"advertisement_id,omitempty"`
	ChannelID       string `bson:"channel_id,omitempty"`

	// زمان‌بندی
	RunAt     time.Time  `bson:"run_at"`
	StartedAt *time.Time `bson:"started_at,omitempty"`
	DoneAt    *time.Time `bson:"done_at,omitempty"`

	// تلاش مجدد
	Attempts    int    `bson:"attempts"`
	MaxAttempts int    `bson:"max_attempts"`
	LastError   string `bson:"last_error,omitempty"`

	// payload اضافی (JSON)
	Payload string `bson:"payload,omitempty"`

	CreatedAt time.Time `bson:"created_at"`
}

// ── Reservation ──────────────────────────────────────────────────

// ReservationStatus وضعیت رزرو.
type ReservationStatus string

const (
	ReservationPending   ReservationStatus = "pending"
	ReservationConfirmed ReservationStatus = "confirmed"
	ReservationSent      ReservationStatus = "sent"
	ReservationCancelled ReservationStatus = "cancelled"
	ReservationFailed    ReservationStatus = "failed"
	// ReservationExpired یعنی چرخه‌ی نمایش (پست اصلی + ریپلی‌ها) به‌طور
	// طبیعی یا به‌خاطر پایان بازه‌ی روزانه (قطع زودهنگام) به آخر رسیده و
	// پیام‌های زنده‌اش پاک شده‌اند.
	ReservationExpired ReservationStatus = "expired"
)

// Reservation رزرو یک اسلات زمانی برای ارسال تبلیغ در یک کانال.
type Reservation struct {
	ID              string            `bson:"_id"`
	InstanceID      string            `bson:"instance_id"`
	CampaignID      string            `bson:"campaign_id"`
	AdvertisementID string            `bson:"advertisement_id"`
	ChannelID       string            `bson:"channel_id"`
	Status          ReservationStatus `bson:"status"`

	// زمان ارسال
	ScheduledAt time.Time  `bson:"scheduled_at"`
	SentAt      *time.Time `bson:"sent_at,omitempty"`

	// LiveMessageIDs شناسه‌ی پیام‌هایی که *همین الان* در کانال زنده و
	// قابل‌مشاهده‌اند (پست اصلی + حداکثر یک ریپلیِ در حال نمایش). هر بار
	// که ریپلی جدید جای قبلی را می‌گیرد یا چرخه پاک می‌شود، این لیست
	// به‌روزرسانی می‌شود؛ برای قطعِ فوریِ یک چرخه (مثلاً پایان بازه‌ی
	// روزانه) همین‌ها حذف می‌شوند.
	LiveMessageIDs []int `bson:"live_message_ids,omitempty"`
	// CurrentReplyIndex ایندکس ریپلیِ در حال نمایش در Advertisement.Replies
	// است؛ -1 یعنی هنوز هیچ ریپلی‌ای نمایش داده نشده (فقط پست اصلی).
	CurrentReplyIndex int `bson:"current_reply_index"`
	// PostedMessageIDs تاریخچه‌ی کامل و append-only همه‌ی پیام‌هایی است که
	// تا الان برای این رزرو ارسال شده (برای آمار/پاک‌سازی ایمن).
	PostedMessageIDs []int      `bson:"posted_message_ids,omitempty"`
	DeleteAt         *time.Time `bson:"delete_at,omitempty"` // زمان حذف خودکار پایان چرخه
	Error            string     `bson:"error,omitempty"`

	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// ── Template ─────────────────────────────────────────────────────

// CampaignTemplate قالب ذخیره‌شده برای ایجاد سریع کمپین توسط ادمین.
type CampaignTemplate struct {
	ID         string `bson:"_id"`
	InstanceID string `bson:"instance_id"`

	Name        string `bson:"name"`
	Description string `bson:"description"`

	// تنظیمات پیش‌فرض کمپین
	StartHour          int `bson:"start_hour"`
	StartMinute        int `bson:"start_minute"`
	EndHour            int `bson:"end_hour"`
	EndMinute          int `bson:"end_minute"`
	IntervalMinutes    int `bson:"interval_minutes"`
	DeleteAfterMinutes int `bson:"delete_after_minutes"`
	RotationMinutes    int `bson:"rotation_minutes"`

	// هدف‌گذاری پیش‌فرض
	TargetTagIDs   []string `bson:"target_tag_ids"`
	MinMemberCount int      `bson:"min_member_count"`

	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// ── Settings ─────────────────────────────────────────────────────

// BotSettings تنظیمات کلی ربات.
type BotSettings struct {
	InstanceID string `bson:"_id"` // instance_id به عنوان _id

	// پیام‌های پیش‌فرض
	WelcomeMessage  string `bson:"welcome_message"`
	HelpMessage     string `bson:"help_message"`
	ContactUsername string `bson:"contact_username"` // پشتیبانی

	// زمان کاری پیش‌فرض (برای کمپین‌های جدید)
	DefaultStartHour int `bson:"default_start_hour"`
	DefaultEndHour   int `bson:"default_end_hour"`

	// ReminderMinutesBefore چند دقیقه پیش از هر ارسال واقعی، پیام یادآوری
	// به ادمین فرستاده شود (۰ = یادآوری خاموش). پیش‌فرض ۱۰.
	ReminderMinutesBefore int `bson:"reminder_minutes_before"`

	UpdatedAt time.Time `bson:"updated_at"`
}

// ── AuditLog ─────────────────────────────────────────────────────

// AuditAction نوع action در لاگ.
type AuditAction string

const (
	AuditCampaignCreate AuditAction = "campaign.create"
	AuditCampaignPause  AuditAction = "campaign.pause"
	AuditCampaignResume AuditAction = "campaign.resume"
	AuditCampaignEnd    AuditAction = "campaign.end"
	AuditAdSend         AuditAction = "ad.send"
	AuditAdFailed       AuditAction = "ad.failed"
	AuditChannelAdd     AuditAction = "channel.add"
	AuditChannelRemove  AuditAction = "channel.remove"
)

// AuditLog ثبت تمام عملیات مهم.
type AuditLog struct {
	ID         string      `bson:"_id"`
	InstanceID string      `bson:"instance_id"`
	Action     AuditAction `bson:"action"`

	ActorID       int64  `bson:"actor_id"` // TelegramID ادمین
	ActorUsername string `bson:"actor_username"`

	TargetID   string `bson:"target_id,omitempty"`   // campaign_id / channel_id / ...
	TargetType string `bson:"target_type,omitempty"` // campaign | channel

	Description string `bson:"description"`
	Extra       string `bson:"extra,omitempty"` // JSON

	CreatedAt time.Time `bson:"created_at"`
}

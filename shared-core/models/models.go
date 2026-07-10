// Package models defines the shared database schema for botmanager, apimanager,
// and agentmanager. All three services import this package — schema changes
// happen in one place and reflect everywhere automatically.
package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base is embedded in every model.
//
// (۲۰۲۶-۰۷-۰۳) json tag های snake_case اضافه شد: قبلاً هیچ‌کدام از مدل‌های این فایل
// json tag نداشتند، یعنی هر endpoint ای که مستقیماً یک struct مدل را (نه یک gin.H
// دستی) برمی‌گرداند، در واقع PascalCase (نام فیلد Go) سریالایز می‌کرد، نه snake_case —
// برخلاف الگوی دستیِ همه‌ی پاسخ‌های gin.H موجود در apimanager. چون apimanager تا امروز
// «کم‌استفاده» بود (PROJECT_UNDERSTANDING بخش ۲) و اولین مصرف‌کننده‌ی واقعی این
// serialization همین وب‌پنل تازه‌ساخته‌شده است، الان بهترین (و آخرین) فرصت برای این
// اصلاح بود؛ تغییر فقط روی encoding/json اثر می‌گذارد، نه رفتار Go/gorm.
type Base struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (b *Base) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// ---- User ----

type UserRole string

const (
	RoleOwner UserRole = "owner"
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	Base
	TelegramID int64    `json:"telegram_id" gorm:"uniqueIndex;not null"`
	Username   string   `json:"username"`
	FirstName  string   `json:"first_name"`
	Role       UserRole `json:"role" gorm:"default:'user'"`
	Balance    float64  `json:"balance" gorm:"default:0"`
	IsBlocked  bool     `json:"is_blocked" gorm:"default:false"`
}

// ---- Server ----

// Server is a remote node running agentmanager and hosting bot containers.
type Server struct {
	Base
	Name     string    `json:"name" gorm:"not null"`
	IP       string    `json:"ip" gorm:"uniqueIndex;not null"`
	IsOnline bool      `json:"is_online" gorm:"default:false"`
	LastSeen time.Time `json:"last_seen"`
	// Channel is the Centrifugo channel this server listens on: "server_<id>"
	Channel string `json:"channel"`

	// OnlineSince آخرین باری که این سرور از offline به online رفت — خودِ apimanager این
	// transition را مدیریت می‌کند (نه agentmanager)، پس همیشه در دسترس است و برای محاسبه‌ی
	// «چقدر آنلاین بوده» واقعی است، نه یک تخمین.
	OnlineSince *time.Time `json:"online_since,omitempty"`

	// CPUPercent/MemoryUsedMB/MemoryTotalMB فقط اگر agentmanager آن‌ها را در heartbeat بفرستد
	// پر می‌شوند — نسخه‌ی فعلی agentmanager این‌ها را نمی‌فرستد، پس همیشه nil خواهند بود تا
	// وقتی agentmanager ارتقا پیدا کند. nil یعنی «گزارش نشده»، نه صفر واقعی.
	CPUPercent    *float64 `json:"cpu_percent,omitempty"`
	MemoryUsedMB  *int64   `json:"memory_used_mb,omitempty"`
	MemoryTotalMB *int64   `json:"memory_total_mb,omitempty"`

	// LastContainers آخرین اسنپ‌شاتِ JSON از وضعیتِ containerهای این سرور طبق heartbeat
	// (name/image/state/status، همان چیزی که از قبل در HeartbeatMsg می‌آمد ولی هیچ‌وقت ذخیره
	// نمی‌شد). json:"-" چون در پاسخ HTTP جدا و parse‌شده برگردانده می‌شود (به قیاس با
	// BotInstance.EnvOverrides/BotTemplate.ConfigSchema).
	LastContainers string `json:"-" gorm:"type:text"`

	// Tags برچسب‌های دلخواهِ سرور، comma-separated (مثلاً "free" یا "free,eu") — بازخورد
	// کاربر ۲۰۲۶-۰۷-۰۵: «سرور فلان با تگ فری، فقط پنل‌های فری به این سرور بیاد». json:"-"
	// چون در پاسخ HTTP به آرایه parse می‌شود.
	Tags string `json:"-" gorm:"type:text"`

	// MaxContainers سقفِ تعداد container مجاز روی این سرور (۰ = نامحدود) — برای جلوگیریِ
	// oversubscribe شدنِ یک سرور؛ در انتخاب سرورِ مقصد هنگام ساخت instance رعایت می‌شود.
	MaxContainers int `json:"max_containers" gorm:"default:0"`
}

// ---- BotTemplate ----

// BotTemplate is a versioned Docker image that can be deployed as a BotInstance.
type BotTemplate struct {
	Base
	Name        string `json:"name" gorm:"not null"`
	Type        string `json:"type" gorm:"not null"` // uploader | vpn | archive | member
	ImageName   string `json:"image_name" gorm:"not null"`
	ImageTag    string `json:"image_tag" gorm:"not null"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active" gorm:"default:true"`
	IsFree      bool   `json:"is_free" gorm:"default:false"`
	// ConfigSchema آرایه‌ی JSON از فیلدهای قابل‌تنظیم توسط کاربرِ instance این قالب —
	// مثال: [{"key":"CHANNEL_ID","label":"آیدی کانال","type":"string","required":true}].
	// مقادیرِ واقعیِ هر کاربر در BotInstance.EnvOverrides ذخیره می‌شود، نه این‌جا؛ این فیلد
	// فقط تعریفِ «چه چیزی قابل‌تنظیم است» است. خالی یعنی این قالب هیچ تنظیمات کاربرمحوری ندارد.
	ConfigSchema string `json:"-" gorm:"type:text"`
}

// ---- BotInstance ----

type InstanceStatus string

const (
	StatusRunning InstanceStatus = "running"
	StatusStopped InstanceStatus = "stopped"
	StatusPending InstanceStatus = "pending"
	StatusError   InstanceStatus = "error"
	// StatusDeleted یعنی instance حذف نرم شده — دیگر فعال یا قابل‌مدیریت
	// نیست (متفاوت از StatusStopped که هنوز می‌تواند دوباره start شود).
	StatusDeleted InstanceStatus = "deleted"
)

// BotInstance is a deployed container owned by a User, running on a Server.
type BotInstance struct {
	Base
	OwnerID     uuid.UUID `json:"owner_id" gorm:"not null;index"`
	TemplateID  uuid.UUID `json:"template_id" gorm:"not null;index"`
	ServerID    uuid.UUID `json:"server_id" gorm:"not null;index"`
	ContainerID string    `json:"container_id"`
	// ContainerName/BotID/DBSchema — رجوع به کامنتِ زیرِ DBSchema برای باگ واقعی که
	// ۲۰۲۶-۰۷-۰۵ از یک لاگ خطای واقعی پیدا شد و اینجا رفع شد: unique index این سه فیلد
	// قبلاً soft-delete (deleted_at) را نادیده می‌گرفت.
	ContainerName string `json:"container_name" gorm:"uniqueIndex:idx_bot_instances_container_name,where:deleted_at IS NULL"`
	BotToken      string `json:"-"` // AES-256-GCM encrypted — هرگز نباید در JSON برگردد

	// BotID عدد یکتای ربات — از توکن استخراج می‌شود (قبل از ':')
	// این فیلد هرگز تغییر نمی‌کند، حتی اگه توکن عوض شود.
	// instance_id در MongoDB = BotID
	// مثال توکن: 8442959411:AAGOZ...  →  BotID = 8442959411
	BotID int64 `json:"bot_id" gorm:"not null;uniqueIndex:idx_bot_instances_bot_id,where:deleted_at IS NULL"`

	// PlanID پلنی که این instance با آن ساخته شده (می‌تواند خالی باشد
	// اگر از یک قالب رایگان مستقیم ساخته شده، نه از مسیر خرید پلن).
	PlanID *uuid.UUID `json:"plan_id" gorm:"index"`

	// LockMode نوع قفل کانال این instance:
	//   "free"   → قفل کانال خود پلتفرم (تبلیغ رایگان ما)
	//   "rented" → قفل کانالی که کسی برایش اجاره پرداخت کرده (از طریق ads-bot)
	//   "none"   → بدون قفل کانال
	LockMode InstanceLockMode `json:"lock_mode" gorm:"default:'none'"`

	Status    InstanceStatus `json:"status" gorm:"default:'pending'"`
	ExpiresAt *time.Time     `json:"expires_at"`

	// DBSchema — باگ واقعی که ۲۰۲۶-۰۷-۰۵ از یک خطای واقعیِ Postgres پیدا شد:
	// «duplicate key value violates unique constraint "idx_bot_instances_db_schema"»
	// وقتی کاربر یک ربات را حذف می‌کرد و بعد می‌خواست همان توکن را دوباره بسازد. علت:
	// DeleteInstance حذفِ نرم (soft-delete، ستون deleted_at) انجام می‌دهد، ولی
	// unique index قبلیِ این فیلد (و مشابهش روی BotID/ContainerName بالا) شرطِ
	// deleted_at IS NULL نداشت — یعنی حتی یک ردیفِ حذف‌شده هم مقدارِ db_schema را برای
	// همیشه اشغال می‌کرد. چک برنامه‌ای (FindInstanceByBotID، که soft-delete را نادیده
	// می‌گیرد) همیشه درست بود؛ فقط خودِ constraint دیتابیس اشتباه بود. با
	// «where:deleted_at IS NULL» به‌صورت partial unique index تعریف شد — یعنی فقط
	// ردیف‌های زنده باید یکتا باشند، ردیف‌های حذف‌شده دیگر مانع نمی‌شوند.
	//
	// ⚠️ چون AutoMigrate ایندکس‌های از قبل موجود در دیتابیس را عوض نمی‌کند (فقط اگر
	// نبود می‌سازد)، این تغییر روی دیتابیسِ در حال اجرا خودکار اعمال نمی‌شود — باید
	// دستی migrate شود (رجوع پیام همین commit/PR برای SQL دقیق).
	DBSchema     string `json:"db_schema" gorm:"uniqueIndex:idx_bot_instances_db_schema,where:deleted_at IS NULL"`
	EnvOverrides string `json:"-" gorm:"type:text"` // JSON: {"CHANNEL_ID": "123"} — داخلی
}

// InstanceLockMode نوع قفل کانال یک instance.
type InstanceLockMode string

const (
	LockModeFree   InstanceLockMode = "free"
	LockModeRented InstanceLockMode = "rented"
	LockModeNone   InstanceLockMode = "none"
)

// IsFreeLock یعنی این instance بخشی از تبلیغ رایگان خود پلتفرم است.
func (b *BotInstance) IsFreeLock() bool { return b.LockMode == LockModeFree }

// IsRentedLock یعنی قفل این instance به یک کمپین اجاره‌ای وصل است.
func (b *BotInstance) IsRentedLock() bool { return b.LockMode == LockModeRented }

// BotIDFromToken Bot ID را از توکن استخراج می‌کند.
// توکن فرمت "12345678:AAGOZ..." دارد — عدد قبل از ':' Bot ID است.
func BotIDFromToken(token string) (int64, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid token format")
	}
	var id int64
	_, err := fmt.Sscanf(parts[0], "%d", &id)
	return id, err
}

// SchemaName نام schema را از ContainerName می‌سازد.
func (b *BotInstance) SchemaName() string {
	if b.DBSchema != "" {
		return b.DBSchema
	}
	return "inst_" + b.ID.String()[:8]
}

// ---- ConfigSchema helpers ----

// ConfigField یک فیلد قابل‌تنظیم توسط کاربر است که ادمین در ConfigSchema قالب تعریف می‌کند.
type ConfigField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Default  string `json:"default"`
	Required bool   `json:"required"`
}

// ParseConfigSchema آرایه‌ی JSON فیلدهای قابل‌تنظیم را برمی‌گرداند.
// اگر ConfigSchema خالی یا نامعتبر باشد، slice خالی برمی‌گردد.
func (t *BotTemplate) ParseConfigSchema() []ConfigField {
	if t.ConfigSchema == "" {
		return nil
	}
	var fields []ConfigField
	_ = json.Unmarshal([]byte(t.ConfigSchema), &fields)
	return fields
}

// ParseEnvOverrides مقادیر کاربرمحور را از JSON به map برمی‌گرداند.
func (inst *BotInstance) ParseEnvOverrides() map[string]string {
	m := map[string]string{}
	if inst.EnvOverrides == "" {
		return m
	}
	_ = json.Unmarshal([]byte(inst.EnvOverrides), &m)
	return m
}

// SetEnvOverrides map را به JSON تبدیل و در EnvOverrides ذخیره می‌کند.
func (inst *BotInstance) SetEnvOverrides(m map[string]string) {
	b, _ := json.Marshal(m)
	inst.EnvOverrides = string(b)
}

// ---- Plan / Payment ----

type Plan struct {
	Base
	// TemplateID deprecated — پلن دیگر به یک تمپلیت وابسته نیست.
	// محدودیت‌ها در PlanBotLimit به تفکیک نوع ربات تعریف می‌شود.
	TemplateID  *uuid.UUID `json:"template_id,omitempty" gorm:"index;default:null"`
	Name        string     `json:"name"`
	DurationDay int        `json:"duration_day"` // 0 = ابدی
	Price       float64    `json:"price"`        // قیمت به TON
	// MaxBots مجموع کل ربات‌ها (fallback اگر PlanBotLimit نبود)
	MaxBots  int  `json:"max_bots" gorm:"default:1"`
	IsFree   bool `json:"is_free" gorm:"default:false"`
	IsActive bool `json:"is_active" gorm:"default:true"`

	// Limits محدودیت به تفکیک نوع ربات
	Limits []PlanBotLimit `json:"limits" gorm:"foreignKey:PlanID"`
}

// PlanBotLimit حداکثر تعداد instance برای هر نوع ربات در یک پلن.
// مثال: پلن Pro → VPN=5, Uploader=3
type PlanBotLimit struct {
	Base
	PlanID  uuid.UUID `json:"plan_id" gorm:"not null;index;uniqueIndex:idx_plan_bottype"`
	BotType string    `json:"bot_type" gorm:"not null;uniqueIndex:idx_plan_bottype"` // uploader | vpn | archive | member
	MaxBots int       `json:"max_bots" gorm:"not null;default:1"`
}

// LimitFor حداکثر تعداد ربات از نوع داده‌شده.
//
// قوانین:
//   - اگر رکورد صریح برای این نوع ربات وجود دارد → همان مقدار برمی‌گردد (0 یعنی مسدود).
//   - اگر رکوردی برای این نوع نیست → MaxBots کلی پلن fallback است.
//
// این یعنی برای مسدودکردن صریح یک نوع ربات، باید رکورد با MaxBots=0 ثبت شود.
func (p *Plan) LimitFor(botType string) int {
	for _, l := range p.Limits {
		if l.BotType == botType {
			return l.MaxBots
		}
	}
	// این نوع ربات صراحتاً محدود نشده → از سقف کلی پلن استفاده می‌شود
	return p.MaxBots
}

// TotalLimit مجموع همه limit ها.
func (p *Plan) TotalLimit() int {
	if len(p.Limits) == 0 {
		return p.MaxBots
	}
	total := 0
	for _, l := range p.Limits {
		total += l.MaxBots
	}
	return total
}

// Subscription اشتراک فعال یک کاربر.
type Subscription struct {
	Base
	UserID    uuid.UUID  `json:"user_id" gorm:"not null;index"`
	PlanID    uuid.UUID  `json:"plan_id" gorm:"not null;index"`
	StartedAt time.Time  `json:"started_at" gorm:"not null"`
	ExpiresAt *time.Time `json:"expires_at"` // nil = ابدی
	IsActive  bool       `json:"is_active" gorm:"default:true;index"`
	BotCount  int        `json:"bot_count" gorm:"default:0"` // تعداد ربات‌های فعلی
}

// HasCapacity بررسی می‌کند آیا کاربر می‌تواند ربات جدید بسازد.
func (s *Subscription) HasCapacity(maxBots int) bool {
	if s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt) {
		return false
	}
	return s.BotCount < maxBots
}

type PaymentStatus string

const (
	PaymentPending PaymentStatus = "pending"
	PaymentDone    PaymentStatus = "done"
	PaymentFailed  PaymentStatus = "failed"
)

type Payment struct {
	Base
	UserID   uuid.UUID     `gorm:"not null;index" json:"user_id"`
	PlanID   *uuid.UUID    `gorm:"index" json:"plan_id,omitempty"`
	Amount   float64       `json:"amount"` // مقدار به TON
	Currency string        `gorm:"default:'TON'" json:"currency"`
	Status   PaymentStatus `gorm:"default:'pending'" json:"status"`
	// TON specific
	TxHash      string     `gorm:"uniqueIndex" json:"tx_hash,omitempty"`    // transaction hash
	FromWallet  string     `json:"from_wallet,omitempty"`                   // کیف پول فرستنده
	PaymentURL  string     `json:"payment_url,omitempty"`                   // لینک پرداخت برای کاربر
	InvoiceID   string     `gorm:"uniqueIndex" json:"invoice_id,omitempty"` // شناسه یکتا
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	InstanceID  *uuid.UUID `json:"instance_id,omitempty"`
}

// AllModels returns every model for db.Migrate().
func AllModels() []any {
	return []any{
		&User{},
		&Server{},
		&BotTemplate{},
		&BotInstance{},
		&Plan{},
		&PlanBotLimit{},
		&Payment{},
		&Subscription{},
		&DeployJob{},
		&InviteLink{},
		&AuditLog{},
		&SourceWorkerConfig{},
		&PromoCode{},
		&PromoRedemption{},
	}
}

// ---- PromoCode ----

// PromoCode یک کدِ تخفیف/شارژ است که هر کاربر حداکثر یک‌بار وارد می‌کند و
// AmountTON به‌صورت اعتبار (credit، نه TON واقعی) به کیف پولش اضافه می‌شود —
// از همان مسیر botpay که AdminCreditExecute هم استفاده می‌کند.
type PromoCode struct {
	Base
	Code      string  `gorm:"uniqueIndex;not null" json:"code"`
	AmountTON float64 `gorm:"not null" json:"amount_ton"`
	// MaxUses سقفِ کلِ استفاده (۰ = نامحدود).
	MaxUses   int        `gorm:"default:0" json:"max_uses"`
	UsedCount int        `gorm:"default:0" json:"used_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	CreatedBy int64      `json:"created_by,omitempty"` // TelegramID ادمینی که ساخته
}

func (p *PromoCode) IsExpired() bool {
	return p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt)
}

func (p *PromoCode) IsExhausted() bool {
	return p.MaxUses > 0 && p.UsedCount >= p.MaxUses
}

// IsRedeemable یعنی همین الان قابل استفاده است (فعال، منقضی/تمام‌نشده).
func (p *PromoCode) IsRedeemable() bool {
	return p.IsActive && !p.IsExpired() && !p.IsExhausted()
}

// PromoRedemption یک بار مصرفِ یک کد توسط یک کاربرِ خاص — با unique index
// روی (PromoID, UserID) از redeem دوباره جلوگیری می‌کند.
type PromoRedemption struct {
	Base
	PromoID uuid.UUID `gorm:"not null;index;uniqueIndex:idx_promo_user"`
	UserID  uuid.UUID `gorm:"not null;index;uniqueIndex:idx_promo_user"`
}

// ---- SourceWorkerConfig ----
//
// SourceWorkerConfig is botmanager's source of truth for the source.worker.*
// NATS contract in shared-core/protocol/source_worker.go. botmanager is the
// responder for that contract (source-service, an internal MTProto/UserBot
// automation tool, is the caller) — an admin creates one row per Telegram
// account a source-service worker should activate as, hands the generated
// LicenseKey to whoever configures that worker, and the worker learns its
// WorkerID + Telegram credentials by calling source.worker.register with
// that key.
type SourceWorkerConfig struct {
	Base
	// Label یادداشتِ آزادِ ادمین برای شناسایی (مثلاً "اکانت پشتیبان ۲"؛
	// نمایشی است، در پروتکل استفاده نمی‌شود.
	Label string

	// LicenseKey به اپراتورِ source-service داده می‌شود (نه به کاربر نهایی)؛
	// worker با همین کلید در source.worker.register خودش را فعال می‌کند.
	LicenseKey string `gorm:"uniqueIndex;not null"`
	// WorkerID شناسه‌ای که در پاسخِ register به worker داده می‌شود و در
	// heartbeat/ازش برمی‌گردد — از قبل (زمان ساخت) تولید می‌شود، نه در
	// لحظه‌ی register، چون باید همیشه ثابت بماند.
	WorkerID string `gorm:"uniqueIndex;not null"`

	AppID   int    `gorm:"not null"`
	AppHash string `gorm:"not null"` // AES-256-GCM encrypted (auth.Encrypt)
	Phone   string `gorm:"not null"`
	// SessionKey کلیدی است که خودِ worker با آن session محلی‌اش را رمز
	// می‌کند — اینجا هم در حالت رمزشده (AES-256-GCM) نگه داشته می‌شود و در
	// پاسخِ register به‌صورت رمزگشایی‌شده تحویل داده می‌شود.
	SessionKey string `gorm:"not null"` // AES-256-GCM encrypted

	IsActive bool `gorm:"default:true"`

	// LastHeartbeatAt/LastStatus از source.worker.heartbeat به‌روزرسانی می‌شوند.
	LastHeartbeatAt *time.Time
	LastStatus      string
}

// IsOnline یعنی طی آستانه‌ی داده‌شده heartbeat دریافت شده — دقیقاً مثل
// Server.IsOnline که با heartbeat واقعی مشخص می‌شود، نه یک فلگ دستی.
func (c *SourceWorkerConfig) IsOnline(threshold time.Duration) bool {
	return c.LastHeartbeatAt != nil && time.Since(*c.LastHeartbeatAt) < threshold
}

// ---- InviteLink ----

// BotType نوع ربات قابل ساخت با InviteLink را مشخص می‌کند.
type BotType string

const (
	BotTypeUploader BotType = "uploader"
	BotTypeVPN      BotType = "vpn"
	BotTypeArchive  BotType = "archive"
	BotTypeMember   BotType = "member"
)

// InviteLink یک لینک دعوت یک‌بار مصرف (یا محدود) است که owner می‌سازد.
// کاربر با /start <token> وارد wizard ساخت ربات می‌شود.
type InviteLink struct {
	Base
	Token     string  `gorm:"uniqueIndex;not null"` // UUID کوتاه — توی لینک می‌آد
	BotType   BotType `gorm:"not null"`
	Label     string  // یادداشت خصوصی owner (مثلاً "برای علی")
	MaxUse    int     `gorm:"default:1"` // 0 = نامحدود
	UsedCount int     `gorm:"default:0"`
	ExpiresAt *time.Time
	CreatedBy int64 // TelegramID سازنده
	// بعد از استفاده، instance ساخته‌شده اینجا ذخیره می‌شه
	InstanceID *uuid.UUID `gorm:"type:uuid"`
}

func (l *InviteLink) IsExpired() bool {
	if l.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*l.ExpiresAt)
}

func (l *InviteLink) IsExhausted() bool {
	if l.MaxUse == 0 {
		return false
	}
	return l.UsedCount >= l.MaxUse
}

func (l *InviteLink) IsValid() bool {
	return !l.IsExpired() && !l.IsExhausted()
}

// ---- Deploy Queue ----

// DeployJobStatus وضعیت یک job در صف deploy.
type DeployJobStatus string

const (
	JobPending    DeployJobStatus = "pending"
	JobProcessing DeployJobStatus = "processing"
	JobDone       DeployJobStatus = "done"
	JobFailed     DeployJobStatus = "failed"
)

// DeployJob یک درخواست deploy در صف.
// apimanager این رو می‌سازه، agentmanager پردازش می‌کنه.
type DeployJob struct {
	Base
	InstanceID  uuid.UUID       `gorm:"not null;index"`
	ServerID    uuid.UUID       `gorm:"not null;index"`
	Status      DeployJobStatus `gorm:"default:'pending';index"`
	Priority    int             `gorm:"default:0"` // بالاتر = زودتر
	Attempts    int             `gorm:"default:0"`
	MaxAttempts int             `gorm:"default:3"`
	ScheduledAt time.Time       // زمان مجاز برای پردازش
	StartedAt   *time.Time
	FinishedAt  *time.Time
	Error       string
}

// ── Audit Log ──────────────────────────────────────────────

// AuditAction نوع action در audit log.
type AuditAction string

const (
	AuditCreateInstance AuditAction = "instance.create"
	AuditDeleteInstance AuditAction = "instance.delete"
	AuditStopInstance   AuditAction = "instance.stop"
	AuditStartInstance  AuditAction = "instance.start"
	AuditBuyPlan        AuditAction = "plan.buy"
	AuditBlockUser      AuditAction = "user.block"
	AuditWithdraw       AuditAction = "wallet.withdraw"
	AuditAdminAction    AuditAction = "admin.action"
)

// AuditLog ثبت همه عملیات مهم.
type AuditLog struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	CreatedAt   time.Time `gorm:"index"`
	ActorID     uuid.UUID `gorm:"type:uuid;index"` // کسی که action انجام داد
	ActorRole   string
	Action      AuditAction `gorm:"not null;index"`
	TargetID    string      `gorm:"index"` // instance_id, user_id, ...
	TargetType  string      // instance, user, plan, wallet
	Description string
	IPAddress   string
	Extra       string `gorm:"type:text"` // JSON extra data
}

// Package models defines the shared database schema for botmanager, apimanager,
// and agentmanager. All three services import this package — schema changes
// happen in one place and reflect everywhere automatically.
package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base is embedded in every model.
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
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

// ---- User ----

type UserRole string

const (
	RoleOwner UserRole = "owner"
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	Base
	TelegramID int64    `gorm:"uniqueIndex;not null"`
	Username   string
	FirstName  string
	Role       UserRole `gorm:"default:'user'"`
	Balance    float64  `gorm:"default:0"`
	IsBlocked  bool     `gorm:"default:false"`
}

// ---- Server ----

// Server is a remote node running agentmanager and hosting bot containers.
type Server struct {
	Base
	Name     string `gorm:"not null"`
	IP       string `gorm:"uniqueIndex;not null"`
	IsOnline bool   `gorm:"default:false"`
	LastSeen time.Time
	// Channel is the Centrifugo channel this server listens on: "server_<id>"
	Channel string
}

// ---- BotTemplate ----

// BotTemplate is a versioned Docker image that can be deployed as a BotInstance.
type BotTemplate struct {
	Base
	Name        string `gorm:"not null"`
	Type        string `gorm:"not null"` // uploader | vpn | archive | member
	ImageName   string `gorm:"not null"`
	ImageTag    string `gorm:"not null"`
	Description string
	IsActive    bool `gorm:"default:true"`
	IsFree      bool `gorm:"default:false"`
}

// ---- BotInstance ----

type InstanceStatus string

const (
	StatusRunning InstanceStatus = "running"
	StatusStopped InstanceStatus = "stopped"
	StatusPending InstanceStatus = "pending"
	StatusError   InstanceStatus = "error"
)

// BotInstance is a deployed container owned by a User, running on a Server.
type BotInstance struct {
	Base
	OwnerID       uuid.UUID      `gorm:"not null;index"`
	TemplateID    uuid.UUID      `gorm:"not null;index"`
	ServerID      uuid.UUID      `gorm:"not null;index"`
	ContainerID   string
	ContainerName string         `gorm:"uniqueIndex"`
	BotToken      string         // AES-256-GCM encrypted

	// BotID عدد یکتای ربات — از توکن استخراج می‌شود (قبل از ':')
	// این فیلد هرگز تغییر نمی‌کند، حتی اگه توکن عوض شود.
	// instance_id در MongoDB = BotID
	// مثال توکن: 8442959411:AAGOZ...  →  BotID = 8442959411
	BotID         int64          `gorm:"uniqueIndex;not null"`

	Status        InstanceStatus `gorm:"default:'pending'"`
	ExpiresAt     *time.Time
	DBSchema      string         `gorm:"uniqueIndex"`
	EnvOverrides  string         `gorm:"type:text"` // JSON: {"CHANNEL_ID": "123"}
}

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

// ---- Plan / Payment ----

type Plan struct {
	Base
	TemplateID  uuid.UUID `gorm:"not null;index"`
	Name        string
	DurationDay int       // 0 = ابدی
	Price       float64   // قیمت به TON
	MaxBots     int       `gorm:"default:1"` // حداکثر تعداد ربات
	IsFree      bool      `gorm:"default:false"` // پلن رایگان
	IsActive    bool      `gorm:"default:true"`
}

// Subscription اشتراک فعال یک کاربر.
type Subscription struct {
	Base
	UserID    uuid.UUID  `gorm:"not null;index"`
	PlanID    uuid.UUID  `gorm:"not null;index"`
	StartedAt time.Time  `gorm:"not null"`
	ExpiresAt *time.Time // nil = ابدی
	IsActive  bool       `gorm:"default:true;index"`
	BotCount  int        `gorm:"default:0"` // تعداد ربات‌های فعلی
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
	PaymentPending  PaymentStatus = "pending"
	PaymentDone     PaymentStatus = "done"
	PaymentFailed   PaymentStatus = "failed"
)

type Payment struct {
	Base
	UserID          uuid.UUID     `gorm:"not null;index"`
	PlanID          *uuid.UUID    `gorm:"index"`
	Amount          float64       // مقدار به TON
	Currency        string        `gorm:"default:'TON'"`
	Status          PaymentStatus `gorm:"default:'pending'"`
	// TON specific
	TxHash          string        `gorm:"uniqueIndex"` // transaction hash
	FromWallet      string        // کیف پول فرستنده
	PaymentURL      string        // لینک پرداخت برای کاربر
	InvoiceID       string        `gorm:"uniqueIndex"` // شناسه یکتا
	ConfirmedAt     *time.Time
	InstanceID      *uuid.UUID
}

// AllModels returns every model for db.Migrate().
func AllModels() []any {
	return []any{
		&User{},
		&Server{},
		&BotTemplate{},
		&BotInstance{},
		&Plan{},
		&Payment{},
		&Subscription{},
		&DeployJob{},
		&InviteLink{},
	}
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
	MaxUse    int     `gorm:"default:1"`  // 0 = نامحدود
	UsedCount int     `gorm:"default:0"`
	ExpiresAt *time.Time
	CreatedBy int64   // TelegramID سازنده
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
	InstanceID    uuid.UUID       `gorm:"not null;index"`
	ServerID      uuid.UUID       `gorm:"not null;index"`
	Status        DeployJobStatus `gorm:"default:'pending';index"`
	Priority      int             `gorm:"default:0"` // بالاتر = زودتر
	Attempts      int             `gorm:"default:0"`
	MaxAttempts   int             `gorm:"default:3"`
	ScheduledAt   time.Time       // زمان مجاز برای پردازش
	StartedAt     *time.Time
	FinishedAt    *time.Time
	Error         string
}

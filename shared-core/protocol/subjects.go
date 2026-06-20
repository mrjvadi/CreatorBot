// Package protocol همه NATS subjects و message types را تعریف می‌کند.
//
// معماری:
//   هر bot یک engine کامل دارد — مستقیم به DB وصل است.
//   NATS فقط برای: deploy، heartbeat، رویدادهای cross-service
//
// Subjects:
//   deploy.{server_id}          ← apimanager → agentmanager (deploy bot)
//   agent.{server_id}.heartbeat ← agentmanager → apimanager
//   agent.{server_id}.result    ← agentmanager → apimanager (نتیجه دستور)
//   event.payment.{bot_id}      ← bot → apimanager (پرداخت تأیید شد)
//   event.instance.{bot_id}     ← apimanager → bot (تغییر وضعیت instance)
package protocol

import "fmt"

// ── Streams ────────────────────────────────────────────────

const (
	StreamDeploy = "DEPLOY"  // دستورات deploy/stop/remove
	StreamAgent  = "AGENT"   // heartbeat + نتایج
	StreamEvents = "EVENTS"  // رویدادهای cross-service
)

// ── Subjects ───────────────────────────────────────────────

// DeploySubject دستور deploy/stop/remove به یک agentmanager.
func DeploySubject(serverID string) string {
	return fmt.Sprintf("deploy.%s", serverID)
}

// HeartbeatSubject heartbeat از agentmanager به apimanager.
func HeartbeatSubject(serverID string) string {
	return fmt.Sprintf("agent.%s.heartbeat", serverID)
}

// ResultSubject نتیجه اجرای یک دستور Docker.
func ResultSubject(serverID string) string {
	return fmt.Sprintf("agent.%s.result", serverID)
}

// PaymentEventSubject پرداخت تأیید شده از bot به apimanager.
func PaymentEventSubject(botID int64) string {
	return fmt.Sprintf("event.payment.%d", botID)
}

// InstanceEventSubject تغییر وضعیت instance (از apimanager به bot).
func InstanceEventSubject(botID int64) string {
	return fmt.Sprintf("event.instance.%d", botID)
}

// ── Message Types ──────────────────────────────────────────

type MsgType string

const (
	// deploy
	MsgDeploy  MsgType = "deploy"
	MsgStop    MsgType = "stop"
	MsgRemove  MsgType = "remove"
	MsgStart   MsgType = "start"
	MsgRestart MsgType = "restart"

	// agent
	MsgHeartbeat MsgType = "heartbeat"
	MsgResult    MsgType = "result"

	// events
	MsgPaymentConfirmed MsgType = "payment_confirmed"
	MsgInstanceUpdated  MsgType = "instance_updated"
	MsgInstanceExpired  MsgType = "instance_expired"
)

// ── Deploy Command ─────────────────────────────────────────

// DeployCommand دستوری که apimanager به agentmanager می‌فرستد.
type DeployCommand struct {
	Type          MsgType           `json:"type"`
	ServerID      string            `json:"server_id"`
	ContainerName string            `json:"container_name"`
	ImageName     string            `json:"image_name"`
	ImageTag      string            `json:"image_tag"`
	EnvVars       map[string]string `json:"env_vars"`
	NetworkName   string            `json:"network_name,omitempty"`
	// ContainerID فقط برای stop/remove/restart لازم است
	ContainerID string `json:"container_id,omitempty"`
}

// ── Heartbeat ──────────────────────────────────────────────

// HeartbeatMsg وضعیت سرور که agentmanager هر N ثانیه ارسال می‌کند.
type HeartbeatMsg struct {
	Type       MsgType           `json:"type"`
	ServerID   string            `json:"server_id"`
	Timestamp  int64             `json:"ts"`
	Containers []ContainerStatus `json:"containers"`
}

// ContainerStatus وضعیت یک container.
type ContainerStatus struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`  // running | exited | paused
	Status string `json:"status"` // "Up 2 hours"
}

// ResultMsg نتیجه اجرای یک دستور Docker.
type ResultMsg struct {
	Type          MsgType `json:"type"`
	ServerID      string  `json:"server_id"`
	ContainerName string  `json:"container_name"`
	CommandType   string  `json:"command_type"`
	Success       bool    `json:"success"`
	Output        string  `json:"output,omitempty"`
	Error         string  `json:"error,omitempty"`
	Timestamp     int64   `json:"ts"`
}

// ── Events ─────────────────────────────────────────────────

// PaymentConfirmedEvent پرداخت تأیید شده.
// bot این رویداد را publish می‌کند تا apimanager رکورد را update کند.
type PaymentConfirmedEvent struct {
	Type       MsgType `json:"type"`
	BotID      int64   `json:"bot_id"`
	UserID     int64   `json:"user_id"`
	PlanID     string  `json:"plan_id"`
	Amount     float64 `json:"amount"`
	Gateway    string  `json:"gateway"`
	RefCode    string  `json:"ref_code"`
	Timestamp  int64   `json:"ts"`
}

// InstanceUpdatedEvent تغییر وضعیت instance از apimanager به bot.
// مثلاً انقضا، تمدید، تغییر تنظیمات.
type InstanceUpdatedEvent struct {
	Type       MsgType           `json:"type"`
	BotID      int64             `json:"bot_id"`
	ChangeType string            `json:"change_type"` // expired | renewed | settings_changed
	Data       map[string]any    `json:"data,omitempty"`
	Timestamp  int64             `json:"ts"`
}

// ── Service Provisioning Events ─────────────────────────────

const (
	// ServiceCreationRequested کاربر درخواست ایجاد سرویس داد.
	ServiceCreationRequested = "service.creation.requested"
	// ServiceCreationStarted agentmanager شروع به deploy کرد.
	ServiceCreationStarted = "service.creation.started"
	// ServiceCreationCompleted deploy موفق.
	ServiceCreationCompleted = "service.creation.completed"
	// ServiceCreationFailed deploy ناموفق → refund باید صورت گیرد.
	ServiceCreationFailed = "service.creation.failed"
	// ServiceStatusChanged تغییر وضعیت سرویس.
	ServiceStatusChanged = "service.status.changed"
)

// ServiceProvisionPayload payload رویداد provisioning.
type ServiceProvisionPayload struct {
	InstanceID  string `json:"instance_id"`
	OwnerID     string `json:"owner_id"`
	ServiceType string `json:"service_type"`
	PlanID      string `json:"plan_id"`
	BotToken    string `json:"bot_token,omitempty"`
	ServerID    string `json:"server_id,omitempty"`
	InvoiceCode string `json:"invoice_code,omitempty"`
	AmountNano  int64  `json:"amount_nano,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ══════════════════════════════════════════════════════════════
// Pay subjects (NATS request/reply) — botpay به‌عنوان responder
// همه‌ی سرویس‌ها برای موجودی و پرداخت از این‌ها استفاده می‌کنند.
// هیچ سرویسی مستقیم به DB کیف پول دست نمی‌زند.
// ══════════════════════════════════════════════════════════════

const (
	// SubjPayBalance — گرفتن موجودی یک کاربر (request/reply)
	SubjPayBalance = "pay.balance"
	// SubjPayAuthorize — تأیید دسترسی یک سرویس به حساب کاربر
	SubjPayAuthorize = "pay.authorize"
	// SubjPayDeduct — کسر از حساب (پرداخت)؛ همه سرویس‌ها از این استفاده می‌کنند
	SubjPayDeduct = "pay.deduct"
	// SubjPayCredit — افزودن اعتبار (فقط admin/داخلی)
	SubjPayCredit = "pay.credit"
	// SubjPayTransfer — انتقال بین دو کاربر
	SubjPayTransfer = "pay.transfer"
	// SubjPayQueue — queue group برای load balancing بین instanceهای botpay
	SubjPayQueue = "botpay-workers"
	// SubjWalletUpdated — رویداد تغییر موجودی (botpay → همه سرویس‌ها)
	// کلاینت‌ها با شنیدن این، کش Redis خود را باطل می‌کنند.
	SubjWalletUpdated = "wallet.updated"
)

// WalletUpdatedEvent رویداد تغییر موجودی یک کاربر.
type WalletUpdatedEvent struct {
	TelegramID int64  `json:"telegram_id"`
	Reason     string `json:"reason"` // deposit | payment | refund | transfer
}

// PayCompletedSubject رویداد اتمام پرداخت برای یک سرویس خاص.
// هر سرویس به pay.completed.<service_id> خودش گوش می‌دهد.
func PayCompletedSubject(serviceID string) string {
	return "pay.completed." + serviceID
}

// PayCompletedEvent جزئیات یک پرداخت تکمیل‌شده برای سرویس درخواست‌کننده.
type PayCompletedEvent struct {
	ServiceID  string  `json:"service_id"`
	TelegramID int64   `json:"telegram_id"`
	AmountTON  float64 `json:"amount_ton"`
	Reason     string  `json:"reason"`   // مثلا "plan:pro"
	Ref        string  `json:"ref"`      // شناسه مرجع سرویس
	Metadata   string  `json:"metadata"` // JSON آزاد
	TxID       string  `json:"tx_id"`
	Success    bool    `json:"success"`
}

// ── Request/Response contracts ────────────────────────────────

// PayRequest پایه‌ی همه‌ی درخواست‌های pay — شامل احراز هویت سرویس.
type PayRequest struct {
	ServiceID  string `json:"service_id"`  // مثلا "botmanager", "uploader"
	ServiceKey string `json:"service_key"` // کلید احراز هویت سرویس
	TelegramID int64  `json:"telegram_id"` // کاربر هدف
}

// BalanceRequest درخواست موجودی.
type BalanceRequest struct {
	PayRequest
}

// BalanceResponse پاسخ موجودی (واحدها TON، نه nano).
type BalanceResponse struct {
	TelegramID int64   `json:"telegram_id"`
	TONBalance float64 `json:"ton_balance"`
	Credit     float64 `json:"credit"`
	Total      float64 `json:"total"`
	Frozen     float64 `json:"frozen"`
	TONAddress string  `json:"ton_address"`
	Error      string  `json:"error,omitempty"`
}

// AuthorizeRequest درخواست تأیید دسترسی سرویس به حساب کاربر.
type AuthorizeRequest struct {
	PayRequest
}

// AuthorizeResponse نتیجه‌ی تأیید.
type AuthorizeResponse struct {
	Authorized bool   `json:"authorized"`
	Error      string `json:"error,omitempty"`
}

// DeductRequest درخواست کسر (پرداخت).
type DeductRequest struct {
	PayRequest
	AmountTON      float64 `json:"amount_ton"`
	Reason         string  `json:"reason"`          // مثلا "plan:starter", "vpn:monthly"
	Ref            string  `json:"ref"`             // شناسه مرجع در سرویس درخواست‌کننده
	Metadata       string  `json:"metadata"`        // JSON آزاد برای شفافیت
	IdempotencyKey string  `json:"idempotency_key"` // جلوگیری از کسر دوباره
}

// DeductResponse نتیجه‌ی کسر.
type DeductResponse struct {
	Success    bool    `json:"success"`
	NewBalance float64 `json:"new_balance"`
	Error      string  `json:"error,omitempty"`
}

// ══════════════════════════════════════════════════════════════
// Member subjects (NATS request/reply) — member-bot به‌عنوان
// زیرساخت متمرکز چک عضویت. bot های فرعی به‌جای ادمین‌شدن در هر
// کانال، از این مسیر می‌پرسند «کاربر X عضو کانال Y هست؟».
// ══════════════════════════════════════════════════════════════

const (
	// SubjMemberCheck — درخواست چک عضویت (request/reply)
	SubjMemberCheck = "member.check"
	// SubjMemberQueue — queue group برای load balancing
	SubjMemberQueue = "memberbot-workers"
)

// MemberCheckRequest درخواست چک عضویت یک کاربر در یک کانال.
type MemberCheckRequest struct {
	ChannelID  int64 `json:"channel_id"`
	UserID     int64 `json:"user_id"`
}

// MemberCheckResponse نتیجه‌ی چک عضویت.
type MemberCheckResponse struct {
	IsMember bool   `json:"is_member"`
	Cached   bool   `json:"cached"`
	Error    string `json:"error,omitempty"`
}

// ══════════════════════════════════════════════════════════════
// FreeBot subjects — وقتی botmanager یک instance با LockMode=free
// می‌سازد، ads-bot باید آن را در FreeBotSlot ثبت کند تا بعداً بتواند
// به یک کمپین اجاره‌ای وصلش کند.
// ══════════════════════════════════════════════════════════════

const SubjFreeBotCreated = "freebot.created"

// FreeBotCreatedEvent رویداد ساخت یک instance رایگان.
type FreeBotCreatedEvent struct {
	InstanceID string `json:"instance_id"` // uuid از shared-core/models.BotInstance.ID
	BotID      int64  `json:"bot_id"`
}

// MembershipJoinedEvent — همان payload که member-bot/internal/events/publisher.go
// روی subject "membership.joined" می‌فرستد.
const SubjMembershipJoined = "membership.joined"

type MembershipJoinedEvent struct {
	TelegramID  int64  `json:"telegram_id"`
	CommunityID int64  `json:"community_id"` // chatID تلگرام
	Source      string `json:"source"`       // "organic" | "invite_link"
	InviteHash  string `json:"invite_hash"`
	JoinedAt    int64  `json:"joined_at"`
	Username    string `json:"username"`
}

// SubjConfirmChannelAdmin — وقتی bot فرعی (مثلا uploader) تشخیص داد که در
// کانال هدف ادمین شده، این را به ads-bot اطلاع می‌دهد تا قفل‌کردن شروع شود.
const SubjConfirmChannelAdmin = "ads.confirm_channel_admin"

// ConfirmChannelAdminRequest درخواست تأیید ادمین‌شدن در کانال.
type ConfirmChannelAdminRequest struct {
	BotID int64 `json:"bot_id"`
}

// ConfirmChannelAdminResponse پاسخ ساده موفق/ناموفق.
type ConfirmChannelAdminResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

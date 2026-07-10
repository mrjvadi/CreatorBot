package models

// ── Force Join Channel ────────────────────────────────────────

// LockMode حالت قفل.
const (
	LockMandatory = "mandatory" // اجباری: عضویت لازم است
	LockOptional  = "optional"  // اختیاری: فقط نمایش داده می‌شود
)

// LockKind نوع قفل.
const (
	LockChannel = "channel"
	LockGroup   = "group"
	LockBot     = "bot"
	LockLink    = "link"
)

// ForceJoinChannel یک «قفل» جوین اجباری/اختیاری (کانال/گروه/ربات/لینک).
type ForceJoinChannel struct {
	Base      `bson:",inline"`
	Kind      string `bson:"kind"` // channel|group|bot|link
	Mode      string `bson:"mode"` // mandatory|optional
	ChatID    int64  `bson:"chat_id"`
	Title     string `bson:"title"`
	Username  string `bson:"username"`   // برای t.me/username
	InviteURL string `bson:"invite_url"` // برای کانال خصوصی یا لینک دلخواه

	// قفل ربات
	BotUsername string `bson:"bot_username,omitempty"`
	BotToken    string `bson:"bot_token,omitempty"`

	// حد عضو: با رسیدن JoinedCount به MemberCap، قفل خودکار غیرفعال می‌شود (0=نامحدود)
	MemberCap   int `bson:"member_cap"`
	JoinedCount int `bson:"joined_count"`

	CheckBot  bool `bson:"check_bot"` // بررسی با ربات واسط member-bot
	IsActive  bool `bson:"is_active"`
	SortOrder int  `bson:"sort_order"`
}

// IsMandatory آیا این قفل اجباری است.
func (f *ForceJoinChannel) IsMandatory() bool { return f.Mode == LockMandatory }

// LinkURL بهترین لینک قابل‌نمایش برای این قفل.
func (f *ForceJoinChannel) LinkURL() string {
	if f.InviteURL != "" {
		return f.InviteURL
	}
	if f.BotUsername != "" {
		return "https://t.me/" + f.BotUsername
	}
	if f.Username != "" {
		return "https://t.me/" + f.Username
	}
	return ""
}

// ── Preview Channel ───────────────────────────────────────────

// PreviewChannel کانالی که پیش‌نمایش رسانه‌ها در آن ارسال می‌شود.
type PreviewChannel struct {
	Base     `bson:",inline"`
	ChatID   int64  `bson:"chat_id"`
	Title    string `bson:"title"`
	IsActive bool   `bson:"is_active"`
}

// ── Ad ────────────────────────────────────────────────────────

// Ad تبلیغی که هنگام نمایش رسانه به کاربر نشان داده می‌شود.
type Ad struct {
	Base       `bson:",inline"`
	Title      string `bson:"title"`
	Text       string `bson:"text"`
	ButtonText string `bson:"button_text"`
	ButtonURL  string `bson:"button_url"`
	IsActive   bool   `bson:"is_active"`
	SortOrder  int    `bson:"sort_order"`
}

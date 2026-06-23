// Package state انواع و ثابت‌های ماشینِ حالتِ مکالمه را تعریف می‌کند.
// منطقِ ذخیره/بازیابی (متدهای Handler) در پکیج tgbot است؛ این پکیج فقط typeها.
package state

// Step مرحله‌ی فعلیِ مکالمه‌ی کاربر.
type Step string

const (
	StepIdle Step = ""

	// سرور
	StepServerName Step = "srv:name"
	StepServerIP   Step = "srv:ip"

	// تمپلیت
	StepTmplType  Step = "tmpl:type"
	StepTmplImage Step = "tmpl:image"
	StepTmplTag   Step = "tmpl:tag"
	StepTmplName  Step = "tmpl:name"

	// پلن
	StepPlanTmpl   Step = "plan:tmpl"
	StepPlanName   Step = "plan:name"
	StepPlanDays   Step = "plan:days"
	StepPlanPrice  Step = "plan:price"
	StepPlanLimits Step = "plan:limits"

	// مدیریت کاربر
	StepUserAction Step = "user:action"
	StepPlanSelect Step = "plan:select"

	// wizard ساخت ربات
	StepWizardToken Step = "wiz:token"
	StepLangSelect  Step = "lang:select"

	// جستجو
	StepBotSearch  Step = "bot:search"
	StepUserSearch Step = "user:search"

	// ادمین — افزودن اعتبار
	StepAdminCreditAmount Step = "admin:credit:amount"

	// کیف پول — واریز
	StepWalletTopupAmount Step = "wallet:topup:amount"

	// ادمین — ارسال همگانی
	StepBroadcastText Step = "broadcast:text"

	// ادمین — دپلوی تستی سرویس
	StepAdminTestToken Step = "admin:test:token"
)

// UserState وضعیتِ ذخیره‌شده‌ی کاربر در Redis.
type UserState struct {
	Step Step              `json:"s"`
	Data map[string]string `json:"d,omitempty"`
}

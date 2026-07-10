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
	// جستجوی کاربر با TelegramID از لیستِ ادمین — قبلاً بعد از لیست هیچ
	// state‌ای فعال نمی‌شد، پس تایپِ TelegramID هیچ اتفاقی نمی‌افتاد.
	StepAdminUserSearch Step = "admin:user:search"

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
	// ادمین — فوروارد همگانی: منتظرِ پیامی که ادمین می‌فرستد/فوروارد می‌کند،
	// سپس منتظرِ تأییدِ ارسال به همه.
	StepBroadcastForwardWait    Step = "broadcast:fwd:wait"
	StepBroadcastForwardConfirm Step = "broadcast:fwd:confirm"

	// ادمین — دپلوی تستی سرویس
	StepAdminTestToken Step = "admin:test:token"

	// ادمین — source-service worker (license/تلگرام)
	StepSWAppID   Step = "sw:appid"
	StepSWAppHash Step = "sw:apphash"
	StepSWPhone   Step = "sw:phone"
	StepSWLabel   Step = "sw:label"

	// کاربر — وارد کردن کدِ پروموشن
	StepPromoRedeem Step = "promo:redeem"

	// ادمین — ساختِ کدِ پروموشن
	StepPromoAdminCode    Step = "promo:admin:code"
	StepPromoAdminAmount  Step = "promo:admin:amount"
	StepPromoAdminMaxUses Step = "promo:admin:maxuses"
	StepPromoAdminDays    Step = "promo:admin:days"

	// wizard — تنظیمات اختصاصی: کاربر فیلدهای ConfigSchema را پر می‌کند
	StepWizardConfig Step = "wiz:config"

	// ادمین — ویرایش ConfigSchema یک قالب
	StepTmplSchemaJSON Step = "tmpl:schema:json"
)

// UserState وضعیتِ ذخیره‌شده‌ی کاربر در Redis.
type UserState struct {
	Step Step              `json:"s"`
	Data map[string]string `json:"d,omitempty"`
}

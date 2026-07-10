// Package models — مدل‌های سند MongoDB برای uploader-bot.
//
// این ربات هیچ داده‌ای در PostgreSQL ذخیره نمی‌کند؛ همه‌ی داده‌ها به‌صورت
// سند در MongoDB (از طریق engine.Mongo) نگهداری می‌شوند. هر سند یک
// InstanceID دارد تا داده‌ی هر ربات از بقیه ایزوله بماند (multi-tenant).
//
// شناسه‌ها (ID) به‌صورت رشته‌ی UUID نگهداری می‌شوند تا از پیچیدگی‌های
// ObjectID/uuid-binary در BSON جلوگیری شود و callback dataها ساده بمانند.
package models

import "time"

// Base پایه‌ی همه‌ی اسناد است.
type Base struct {
	ID         string    `bson:"_id"          json:"id"`
	InstanceID string    `bson:"instance_id"  json:"instance_id"`
	CreatedAt  time.Time `bson:"created_at"   json:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at"   json:"updated_at"`
}

// ── Setting ───────────────────────────────────────────────────

// Setting یک تنظیم کلید/مقدار متنی است (collection: settings).
type Setting struct {
	InstanceID string `bson:"instance_id" json:"instance_id"`
	Key        string `bson:"key"         json:"key"`
	Value      string `bson:"value"       json:"value"`
}

// کلیدهای تنظیمات پیش‌فرض.
const (
	// متن‌ها
	SettingWelcomeText     = "welcome_text"
	SettingAdminWelcome    = "admin_welcome"
	SettingNotMemberText   = "not_member_text"
	SettingPasswordText    = "password_text"
	SettingSubRequiredText = "sub_required_text"
	SettingNotFoundText    = "not_found_text"
	SettingSupportText     = "support_text"
	SettingHelpText        = "help_text"
	SettingStartButtons    = "start_buttons" // دکمه‌های شروع پیشرفته: هر خط «برچسب|لینک»

	// وضعیت‌ها (true/false)
	SettingBotActive          = "bot_active"
	SettingSubRequired        = "sub_required" // اشتراک اجباری سراسری
	SettingUserUpload         = "user_upload"  // آپلود توسط کاربر
	SettingAutoApproveFiles   = "auto_approve_files"
	SettingShowSearch         = "show_search"   // جستجو
	SettingInlineSearch       = "inline_search" // جستجوی اینلاین
	SettingShowLikesButtons   = "show_likes"
	SettingShowReportButton   = "show_report"
	SettingShowResendButton   = "show_resend"
	SettingShowComment        = "show_comment"
	SettingForwardLockDefault = "forward_lock_default"
	SettingAntiFilterDefault  = "antifilter_default"
	SettingThumbUploadDefault = "thumb_upload_default"
	SettingVideoThumbDefault  = "video_thumb_default"
	SettingRemoveLinks        = "remove_links" // حذف خودکار لینک از کپشن
	SettingSignatureEnabled   = "signature_enabled"
	SettingForceSeen          = "force_seen"      // سین اجباری فیک
	SettingForceReact         = "force_react"     // ری‌اکشن اجباری فیک
	SettingLeaveReport        = "leave_report"    // گزارش لفت از قفل اجباری به کاربر
	SettingStorageChannel     = "storage_channel" // آیدی کانال خصوصی ذخیره‌سازی فایل‌ها

	// نمایش دکمه‌های کاربری (on/off)
	SettingBtnPopular = "btn_popular"
	SettingBtnNewest  = "btn_newest"
	SettingBtnTop     = "btn_top"
	SettingBtnUpload  = "btn_upload"
	SettingBtnSupport = "btn_support"

	// نام (برچسب) دکمه‌های کاربری — قابل تغییر توسط ادمین
	SettingLblSearch  = "lbl_search"
	SettingLblHelp    = "lbl_help"
	SettingLblSupport = "lbl_support"
	SettingLblPopular = "lbl_popular"
	SettingLblNewest  = "lbl_newest"
	SettingLblTop     = "lbl_top"
	SettingLblUpload  = "lbl_upload"

	// مقادیر عددی/متنی
	SettingFreeDownloads        = "free_downloads"
	SettingAutoDeleteDefault    = "auto_delete_default"    // ثانیه
	SettingSpamDelay            = "spam_delay"             // ثانیه
	SettingBroadcastInterval    = "broadcast_interval"     // دقیقه
	SettingBroadcastPin         = "broadcast_pin"          // پین پیام بعد از ارسال همگانی
	SettingBroadcastAutoDelete  = "broadcast_autodelete"   // حذف خودکار پیام‌های همگانی
	SettingBroadcastDeleteHours = "broadcast_delete_hours" // بعد از چند ساعت حذف شود
	SettingBroadcastDelayMS     = "broadcast_delay_ms"     // فاصله‌ی بین هر ارسال (میلی‌ثانیه)
	SettingSignature            = "signature"
	SettingCodePrefix           = "code_prefix"           // پیشوند اختصاصی کدهای این ربات (خودکار)
	SettingAutoDeleteWarn       = "auto_delete_warn"      // متن هشدار حذف خودکار ({sec} = ثانیه)
	SettingAutoDeleteWarnOff    = "auto_delete_warn_off"  // "true" = اصلاً هشدار نده
	SettingAutoDeleteWarnKeep   = "auto_delete_warn_keep" // "true" = هشدار پاک نشود

	// پرداخت‌ها
	SettingPaymentZarinpal  = "payment_zarinpal"
	SettingPaymentZibal     = "payment_zibal"
	SettingPaymentCard      = "payment_card"
	SettingPaymentTON       = "payment_ton"
	SettingPaymentTRON      = "payment_tron"
	SettingPaymentStars     = "payment_stars"
	SettingActiveGateway    = "active_gateway"
	SettingZarinpalMerchant = "zarinpal_merchant"
	SettingZibalMerchant    = "zibal_merchant"
	SettingCardNumber       = "card_number"
	SettingCardHolder       = "card_holder"
	SettingTONWallet        = "ton_wallet"
	SettingTRONWallet       = "tron_wallet"
)

// ── Backup ────────────────────────────────────────────────────

// Backup متادیتای یک بکاپ ساخته‌شده (فایل بکاپ در تلگرام نگهداری می‌شود).
type Backup struct {
	Base       `bson:",inline"`
	FileID     string `bson:"file_id"`
	FileSize   int64  `bson:"file_size"`
	TotalCodes int    `bson:"total_codes"`
	TotalFiles int    `bson:"total_files"`
	CreatedBy  int64  `bson:"created_by"`
}

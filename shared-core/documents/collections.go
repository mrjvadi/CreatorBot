// Package documents تعریف document های MongoDB و نام collection ها را نگه می‌دارد.
// همه ربات‌ها از این package برای دسترسی به MongoDB استفاده می‌کنند.
// Multi-tenant: هر document یک instance_id دارد.
package documents

// Collection names
const (
	// uploader-bot
	ColCodes    = "codes"
	ColFiles    = "files"
	ColUsers    = "bot_users"

	// vpn-bot
	ColSubscriptions = "subscriptions"
	ColPanels        = "vpn_panels"

	// archive-bot
	ColArchiveFiles = "archive_files"

	// member-bot
	ColVerifications = "member_verifications"
	ColLocks         = "member_locks"

	// مشترک همه ربات‌ها
	ColStats    = "stats"
	ColLogs     = "bot_logs"
	ColSettings = "bot_settings"
)

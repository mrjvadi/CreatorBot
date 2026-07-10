// Package migrations فایل‌های SQL ورژن‌دار هر سرویس را embed می‌کند.
//
// ساختار: migrations/<service>/<NNNN>_<name>.sql
// نسخه‌ی هر migration همان عدد ابتدای اسم فایل است و به‌ترتیب صعودی اجرا
// می‌شود. برای اضافه‌کردن نسخه‌ی جدید از دستور `dbmigrate new` استفاده کنید
// (یا دستی فایل بسازید) — چون embed است، بعد از اضافه‌کردن فایل باید باینری
// دوباره build شود.
package migrations

import "embed"

//go:embed */*.sql
var FS embed.FS

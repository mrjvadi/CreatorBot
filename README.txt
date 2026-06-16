دستورالعمل اعمال fix ها (دور دوم عیب‌یابی)
═══════════════════════════════════════════

۱. uploader-bot — کل پکیج tgbot عوض شده (مهم‌ترین):
   ⚠️ اول فایل‌های قدیمی رو حذف کن:
   rm uploader-bot/internal/tgbot/user.go
   rm uploader-bot/internal/tgbot/admin.go
   
   بعد همه فایل‌های uploader-bot-tgbot/ رو کپی کن به:
   uploader-bot/internal/tgbot/

۲. vpn-bot/user.go → کپی به vpn-bot/internal/tgbot/user.go

۳. i18n/en.go → کپی به botmanager/internal/tgbot/i18n/en.go

۴. پوشه stray رو حذف کن (اگه هست):
   rm -rf botmanager/internal/tgbot/wizard/

۵. build مجدد:
   go clean -cache
   cd uploader-bot && go run cmd/bot/main.go
   cd botmanager   && go run cmd/*.go

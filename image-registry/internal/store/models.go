// Package store مدل‌ها و دسترسی DB سرویس image-registry.
package store

import (
	"time"

	"github.com/google/uuid"
)

// RegisteredImage یک image:tag که اجازه‌ی deploy شدن روی هر سرور پلتفرم را
// دارد — این جدول جایگزین «whitelist محلی» قدیمی agentmanager (env var
// ALLOWED_IMAGES) می‌شود؛ به‌جای اینکه هر agentmanager خودش یک لیست prefix
// داشته باشد، همه از این‌جا (یک منبع حقیقت مرکزی) سؤال می‌کنند.
// json tag ها ۲۰۲۶-۰۷-۰۵ اضافه شدند (بازخورد کاربر: پنل وب اسم/تگ/... ایمیج‌ها را خالی
// نشان می‌داد). قبلاً این struct هیچ json tag ای نداشت — یعنی listImages/createImage
// (که مستقیماً همین struct را json.Marshal می‌کنند، رجوع internal/api/api.go) فیلدها را
// با نام فیلدِ Go (PascalCase: Name، Tag، ServiceType، IsActive، ...) برمی‌گرداندند، نه
// snake_case که هر مصرف‌کننده‌ی HTTP طبیعی انتظارش را دارد. تنها مسیرِ حیاتیِ
// agentmanager (GET /v1/check) از این struct مستقیم استفاده نمی‌کند — یک gin.H دستی با
// کلیدهای snake_case درست می‌سازد — پس این تغییر روی آن مسیر هیچ اثری ندارد؛ فقط
// روی پاسخ‌های خام‌ِ لیست/ثبت که پنل ادمین می‌خواند.
type RegisteredImage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Name نام کامل image بدون تگ — مثلاً "creatorbot/uploader-bot".
	Name string `gorm:"not null;uniqueIndex:idx_name_tag" json:"name"`
	// Tag نسخه — مثلاً "v1.4.0" یا "latest".
	Tag string `gorm:"not null;uniqueIndex:idx_name_tag" json:"tag"`

	// ServiceType اختیاری — نوع سرویس مطابق shared-core/models.BotTemplate.Type
	// (uploader|vpn|archive|member|...) فقط برای خوانایی/فیلتر در پنل ادمین،
	// در تصمیم مجاز/غیرمجاز بودن نقشی ندارد.
	ServiceType string `json:"service_type,omitempty"`
	Description string `json:"description,omitempty"`

	IsActive bool `gorm:"default:true;index" json:"is_active"`

	// ── آرتیفکت واقعی image ────────────────────────────────
	// این سرویس دیگر فقط یک whitelist از نام‌ها نیست — خودِ فایل image
	// (خروجی `docker save name:tag > file.tar`) این‌جا آپلود و نگه‌داری
	// می‌شود، تا agentmanager به‌جای `docker pull` از یک registry بیرونی،
	// مستقیماً این فایل را از خودِ image-registry دانلود و `docker load`
	// کند. رجوع README برای جزئیات و توجیه.

	// FilePath مسیر فایل روی دیسک سرویس (داخل IMAGE_STORAGE_DIR) — عمداً هرگز در
	// پاسخ HTTP برنمی‌گردد (یک مسیر فایل داخلیِ سرور است، نه چیزی که کلاینت لازم
	// داشته باشد). خالی یعنی هنوز فایلی آپلود نشده.
	FilePath string `json:"-"`
	// FileSHA256 چک‌سام فایل آپلودشده — agentmanager باید بعد از دانلود
	// این را با فایل دریافتی مقایسه کند تا از سالم بودن انتقال مطمئن شود.
	// خالی بودن این فیلد یعنی «هنوز فایلی آپلود نشده» (معادلِ HasFile()==false).
	FileSHA256 string `json:"file_sha256,omitempty"`
	// FileSize اندازه‌ی فایل به بایت (برای نمایش/گزارش، نه اعتبارسنجی خودِ محتوا).
	FileSize int64 `json:"file_size,omitempty"`
}

// FullRef مثل چیزی که agentmanager می‌سازد: "name:tag".
func (r *RegisteredImage) FullRef() string { return r.Name + ":" + r.Tag }

// HasFile یعنی آیا فایل واقعی image برای این ردیف آپلود شده یا نه.
func (r *RegisteredImage) HasFile() bool { return r.FilePath != "" }

// AllowedCaller یک IP/CIDR (و اختیاراً یک دامنه) که اجازه دارد از این سرویس
// سؤال کند. این دقیقاً همان چیزی است که کاربر خواسته: «whitelist بر اساس IP»
// — نه فقط لیست ایمیج، بلکه لیست اینکه *کی* اجازه دارد بپرسد/ثبت کند.
// json tag ها ۲۰۲۶-۰۷-۰۵ اضافه شدند — رجوع به کامنتِ مشابه روی RegisteredImage در همین فایل.
type AllowedCaller struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Label نام خوانا — مثلاً "server-1-frankfurt" یا "ci-runner".
	Label string `gorm:"not null" json:"label"`

	// CIDR شکل IP/CIDR — تک IP را هم به‌صورت "1.2.3.4/32" بنویسید.
	// این تنها راه واقعی احراز است (رجوع به README برای محدودیت‌های Domain).
	CIDR string `gorm:"not null" json:"cidr"`

	// Domain اختیاری — اگر ست شود، سرویس در لحظه‌ی هر درخواست این دامنه را
	// resolve می‌کند و IP فراخوان باید هم در CIDR بالا باشد هم در نتیجه‌ی
	// resolve این دامنه — یک لایه‌ی دوم، نه جایگزین CIDR (چون DNS به‌تنهایی
	// قابل‌جعل/ناپایدار است؛ رجوع به README بخش «محدودیت‌های شناخته‌شده»).
	Domain string `json:"domain,omitempty"`

	// CanWrite یعنی این caller اجازه‌ی ثبت/حذف image یا مدیریت callerهای
	// دیگر را هم دارد؛ false یعنی فقط اجازه‌ی GET /v1/check و GET /v1/images
	// (یعنی همان چیزی که خودِ کاربر خواست: «فقط بتواند فلان کار را بکند»).
	CanWrite bool `gorm:"default:false" json:"can_write"`

	IsActive bool `gorm:"default:true;index" json:"is_active"`
}

package models

import "time"

// ── Folder / Category ─────────────────────────────────────────

// Folder یک پوشه یا زیرپوشه است. ParentID خالی یعنی پوشه‌ی ریشه.
//
// json tag ها ۲۰۲۶-۰۷-۰۵ اضافه شدند — همان باگِ کلاسیِ «مدل بدون json tag، فقط bson»
// که قبلاً در shared-core/models پیدا شد این‌جا هم بود: بدون این‌ها، جواب NATS به
// apimanager (که این‌ها را json.Marshal می‌کند، نه bson) با نام فیلدهای PascalCase
// برمی‌گشت، نه snake_case.
type Folder struct {
	Base      `bson:",inline"`
	Name      string `bson:"name" json:"name"`
	ParentID  string `bson:"parent_id" json:"parent_id"` // "" = ریشه
	Icon      string `bson:"icon" json:"icon,omitempty"`
	SortOrder int    `bson:"sort_order" json:"sort_order"`
	IsActive  bool   `bson:"is_active" json:"is_active"`
}

// ── Media Code ────────────────────────────────────────────────

type CodeType string

const (
	CodeOnce      CodeType = "once"
	CodeLimited   CodeType = "limited"
	CodeUnlimited CodeType = "unlimited"
	CodeExpiry    CodeType = "expiry"
)

// Code یک «کد رسانه» است که یک یا چند فایل را به کاربر تحویل می‌دهد.
//
// json tag ها ۲۰۲۶-۰۷-۰۵ اضافه شدند — رجوع به کامنتِ مشابه روی Folder در همین فایل.
type Code struct {
	Base     `bson:",inline"`
	Code     string   `bson:"code" json:"code"` // کد رسانه (قابل تغییر)
	Type     CodeType `bson:"type" json:"type"`
	FolderID string   `bson:"folder_id" json:"folder_id"` // "" = بدون پوشه

	// محتوا
	Caption   string `bson:"caption" json:"caption,omitempty"`
	Thumbnail string `bson:"thumbnail" json:"thumbnail,omitempty"` // file_id تامبنیل
	IsAlbum   bool   `bson:"is_album" json:"is_album"`

	// محدودیت‌ها
	MaxUse    int        `bson:"max_use" json:"max_use"`
	UsedCount int        `bson:"used_count" json:"used_count"`
	ExpiresAt *time.Time `bson:"expires_at,omitempty" json:"expires_at,omitempty"`

	// قفل‌ها / تنظیمات تکی
	ForwardLock   bool   `bson:"forward_lock" json:"forward_lock"`
	AntiFilter    bool   `bson:"anti_filter" json:"anti_filter"`   // ضدفیلتر تکی
	ChannelLock   bool   `bson:"channel_lock" json:"channel_lock"` // جوین اجباری
	AutoDelete    int    `bson:"auto_delete" json:"auto_delete"`   // ثانیه (0=غیرفعال)
	Password      string `bson:"password" json:"-"`                // هرگز نباید در پاسخ HTTP/NATS برگردد
	DownloadLimit int    `bson:"download_limit" json:"download_limit"`

	// آمار فیک
	FakeLikes     int `bson:"fake_likes" json:"fake_likes"`
	FakeDownloads int `bson:"fake_downloads" json:"fake_downloads"`
	FakeViews     int `bson:"fake_views" json:"fake_views"`

	// گیت‌های فیک قبل از دانلود
	ForceSeen  bool `bson:"force_seen" json:"force_seen"`   // سین اجباری فیک
	ForceReact bool `bson:"force_react" json:"force_react"` // ری‌اکشن اجباری فیک

	// اشتراک
	SubRequired bool `bson:"sub_required" json:"sub_required"`

	// در انتظار تایید ادمین (آپلود کاربر، وقتی تایید خودکار خاموش است)
	Pending bool `bson:"pending" json:"pending"`

	UploaderID int64 `bson:"uploader_id" json:"uploader_id,omitempty"` // telegram_id آپلودکننده

	// شناسه‌ی فایل‌ها به ترتیب (به‌جای جدول واسط). برای سازگاری
	// متدهای GetFilesForCode/AddFileToCode/RemoveFileFromCode هم نگه داشته شده‌اند.
	FileIDs []string `bson:"file_ids" json:"file_ids,omitempty"`

	// انواع رسانه‌ی این کد (photo/video/...) برای جستجو بر اساس نوع.
	MediaTypes []string `bson:"media_types" json:"media_types,omitempty"`
}

// Entity یک بازهٔ قالب‌بندی در کپشن (بولد، لینک، …) — برای حفظ فرمت اصلی.
type Entity struct {
	Type     string `bson:"type"`
	Offset   int    `bson:"offset"`
	Length   int    `bson:"length"`
	URL      string `bson:"url,omitempty"`
	Language string `bson:"language,omitempty"`
}

// File یک فایل تلگرام.
type File struct {
	Base            `bson:",inline"`
	FileID          string   `bson:"file_id"`
	FileType        string   `bson:"file_type"` // video|photo|audio|document|animation|voice|sticker
	Caption         string   `bson:"caption"`
	CaptionEntities []Entity `bson:"caption_entities,omitempty"` // قالب‌بندی/هایپرلینک
	Thumbnail       string   `bson:"thumbnail"`                  // file_id کاور (برای ویدیو)
	UploaderID      int64    `bson:"uploader_id"`
	SourceUUID      string   `bson:"source_uuid,omitempty"`

	// مرجع پایدار در کانال ذخیره‌سازی (برای مقاومت در برابر تغییر توکن/مهاجرت)
	StorageChatID int64 `bson:"storage_chat_id,omitempty"`
	StorageMsgID  int   `bson:"storage_msg_id,omitempty"`
}

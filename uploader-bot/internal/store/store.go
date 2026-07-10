// Package store — لایه‌ی داده‌ی uploader-bot روی MongoDB.
//
// این ربات هیچ داده‌ای در PostgreSQL نمی‌نویسد. همه‌ی repositoryها از
// engine.Mongo (ports.DocumentStore) استفاده می‌کنند و هر کوئری به‌صورت
// خودکار با instance_id فیلتر می‌شود تا داده‌ی هر ربات ایزوله بماند.
//
// کد به‌صورت ماژولار، فایل‌به‌فایل برای هر دامنه تقسیم شده است:
//
//	user.go, code.go, file.go, folder.go, plan.go, payment.go,
//	channel.go, admin.go, setting.go, stats.go, backup.go, download.go, ads.go
package store

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// نام collectionها.
const (
	colSettings  = "settings"
	colUsers     = "users"
	colCodes     = "codes"
	colFiles     = "files"
	colFolders   = "folders"
	colForceJoin = "force_join_channels"
	colPreview   = "preview_channels"
	colSubPlans  = "sub_plans"
	colPayments  = "payments"
	colBackups   = "backups"
	colDownloads = "download_logs"
	colAdmins    = "admins"
	colAds       = "ads"
)

// Store مخزن داده روی MongoDB با کش اختیاری روی Redis.
type Store struct {
	ds         ports.DocumentStore
	instanceID string
	cache      ports.Cache  // اختیاری؛ برای کش تحویل کد (کاهش درخواست به DB)
	log        ports.Logger // اختیاری؛ اگر nil باشد، خطاهای best-effort فقط بی‌صدا نادیده گرفته می‌شوند

	// settingsCache کش درون‌حافظه‌ایِ تنظیمات — چون GetSetting در تحویل هر
	// کد چندین‌بار صدا زده می‌شود (رمز/اشتراک/امضا/حذف‌خودکار/...)، بدون این
	// کش هر تحویل یک فایل به ۱۰+ رفت‌وبرگشت جداگانه به Mongo نیاز داشت.
	// روی SetSetting (چه از پنل، چه از NATS) بلافاصله invalidate/به‌روز می‌شود،
	// پس نیازی به TTL نیست — همیشه با آخرین مقدار نوشته‌شده هماهنگ است.
	settingsMu    sync.RWMutex
	settingsCache map[string]string
}

// New یک Store جدید می‌سازد. ds از engine.Mongo، instanceID از engine.InstanceID،
// cache از engine.Cache (می‌تواند nil باشد)، و log از engine.Log (می‌تواند nil باشد
// — در آن صورت خطاهای عملیات‌های best-effort فقط لاگ نمی‌شوند، نادیده گرفته نمی‌شوند
// چون این متدها همچنان err را برمی‌گردانند؛ فقط لاگ داخلی غیرفعال است).
func New(ds ports.DocumentStore, instanceID string, cache ports.Cache, log ports.Logger) *Store {
	return &Store{
		ds:            ds,
		instanceID:    instanceID,
		cache:         cache,
		log:           log,
		settingsCache: make(map[string]string),
	}
}

// logErr خطای عملیات‌های best-effort (که نمی‌توانند/نباید جلوی جریان اصلی را
// بگیرند — مثل نامعتبرسازی کش یا آمار) را به‌جای نادیده‌گرفتن کامل، لاگ می‌کند.
func (s *Store) logErr(op string, err error) {
	if err == nil {
		return
	}
	if s.log != nil {
		s.log.Error("store: "+op, ports.F("err", err))
	}
}

// col یک collection با نام مشخص برمی‌گرداند.
func (s *Store) col(name string) ports.Collection { return s.ds.Collection(name) }

// f یک filter می‌سازد که همیشه instance_id را در بر دارد.
func (s *Store) f(extra ...bson.E) bson.D {
	d := bson.D{{Key: "instance_id", Value: s.instanceID}}
	return append(d, extra...)
}

// set یک bson update از نوع $set با updated_at می‌سازد.
func set(fields bson.D) bson.D {
	fields = append(fields, bson.E{Key: "updated_at", Value: time.Now()})
	return bson.D{{Key: "$set", Value: fields}}
}

// newID یک شناسه‌ی رشته‌ای یکتا می‌سازد.
func newID() string { return uuid.New().String() }

// ── Find option helpers ───────────────────────────────────────

func ports_sortDesc(field string) ports.FindOption {
	return ports.WithSort(bson.D{{Key: field, Value: -1}})
}

func ports_sortAsc(field string) ports.FindOption {
	return ports.WithSort(bson.D{{Key: field, Value: 1}})
}

func ports_skip(n int64) ports.FindOption  { return ports.WithSkip(n) }
func ports_limit(n int64) ports.FindOption { return ports.WithLimit(n) }

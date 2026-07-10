// Package store — لایه‌ی داده‌ی admanager-bot روی MongoDB.
//
// همه‌ی کوئری‌ها با instance_id فیلتر می‌شوند تا داده‌ی هر ربات
// از ربات‌های دیگر ایزوله بماند. الگوی مشابه uploader-bot.
package store

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// نام collectionها.
const (
	colChannels     = "channels"
	colTags         = "tags"
	colCampaigns    = "campaigns"
	colAds          = "advertisements"
	colJobs         = "scheduled_jobs"
	colReservations = "reservations"
	colTemplates    = "campaign_templates"
	colSettings     = "settings"
	colAuditLogs    = "audit_logs"
)

// Store مخزن داده روی MongoDB.
type Store struct {
	ds         ports.DocumentStore
	instanceID string
	cache      ports.Cache // اختیاری
}

// New یک Store جدید می‌سازد.
func New(ds ports.DocumentStore, instanceID string, cache ports.Cache) *Store {
	return &Store{ds: ds, instanceID: instanceID, cache: cache}
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

func sortDesc(field string) ports.FindOption {
	return ports.WithSort(bson.D{{Key: field, Value: -1}})
}

func sortAsc(field string) ports.FindOption {
	return ports.WithSort(bson.D{{Key: field, Value: 1}})
}

func skip(n int64) ports.FindOption  { return ports.WithSkip(n) }
func limit(n int64) ports.FindOption { return ports.WithLimit(n) }

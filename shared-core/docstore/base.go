// Package docstore repository های MongoDB با multi-tenant filtering را فراهم می‌کند.
// هر method به‌صورت خودکار instance_id را به filter اضافه می‌کند.
package docstore

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Base پایه همه repository های MongoDB است.
// instanceID: UUID ربات از جدول bot_instances در PostgreSQL.
type Base struct {
	ds         ports.DocumentStore
	instanceID string
}

func NewBase(ds ports.DocumentStore, instanceID string) Base {
	return Base{ds: ds, instanceID: instanceID}
}

// col یک collection با نام مشخص برمی‌گرداند.
func (b *Base) col(name string) ports.Collection {
	return b.ds.Collection(name)
}

// baseFilter یک bson.D با instance_id می‌سازد و فیلترهای اضافی رو append می‌کند.
func (b *Base) baseFilter(extra ...bson.E) bson.D {
	d := bson.D{{Key: "instance_id", Value: b.instanceID}}
	return append(d, extra...)
}

// newDocBase یک DocBase جدید با instance_id و timestamp می‌سازد.
func (b *Base) newDocBase() ports.DocBase {
	now := time.Now()
	return ports.DocBase{
		InstanceID: b.instanceID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// setUpdate یک $set update با updated_at می‌سازد.
func setUpdate(fields bson.D) bson.D {
	fields = append(fields, bson.E{Key: "updated_at", Value: time.Now()})
	return bson.D{{Key: "$set", Value: fields}}
}

// SettingStore تنظیمات متنی ربات.
type SettingStore struct {
	Base
}

func NewSettingStore(ds ports.DocumentStore, instanceID string) *SettingStore {
	return &SettingStore{Base: NewBase(ds, instanceID)}
}

func (s *SettingStore) Get(ctx context.Context, key string) (string, error) {
	type doc struct {
		Value string `bson:"value"`
	}
	var d doc
	err := s.col("bot_settings").FindOne(ctx,
		s.baseFilter(bson.E{Key: "key", Value: key}), &d)
	if err != nil {
		return "", nil // key وجود نداره — مقدار پیش‌فرض
	}
	return d.Value, nil
}

func (s *SettingStore) Set(ctx context.Context, key, value string) error {
	filter := s.baseFilter(bson.E{Key: "key", Value: key})
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "value", Value: value},
			{Key: "updated_at", Value: time.Now()},
			{Key: "instance_id", Value: s.instanceID},
			{Key: "key", Value: key},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "created_at", Value: time.Now()},
		}},
	}
	// upsert
	col := s.ds.Collection("bot_settings")
	return col.UpdateOne(ctx, filter, update)
}

// StatStore آمار روزانه ربات.
type StatStore struct {
	Base
}

func NewStatStore(ds ports.DocumentStore, instanceID string) *StatStore {
	return &StatStore{Base: NewBase(ds, instanceID)}
}

func (s *StatStore) IncrementDaily(ctx context.Context, field string, delta int64) error {
	date := time.Now().Format("2006-01-02")
	filter := s.baseFilter(bson.E{Key: "date", Value: date})
	update := bson.D{
		{Key: "$inc", Value: bson.D{{Key: field, Value: delta}}},
		{Key: "$set", Value: bson.D{
			{Key: "updated_at", Value: time.Now()},
			{Key: "instance_id", Value: s.instanceID},
			{Key: "date", Value: date},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "created_at", Value: time.Now()},
		}},
	}
	return s.col("stats").UpdateOne(ctx, filter, update)
}

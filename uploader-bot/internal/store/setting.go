package store

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// GetSetting مقدار یک تنظیم را برمی‌گرداند. اگر کلید نباشد، رشته‌ی خالی.
// ابتدا از کش درون‌حافظه‌ای خوانده می‌شود (پر می‌شود از SetSetting/NATS)؛ فقط در
// نبود مقدار در کش به Mongo مراجعه می‌شود — چون این تابع در تحویل هر فایل چندین
// بار صدا زده می‌شود (رمز، اشتراک، امضا، حذف‌خودکار، ...).
func (s *Store) GetSetting(ctx context.Context, key string) string {
	if v, ok := s.settingsCacheGet(key); ok {
		return v
	}
	var st models.Setting
	err := s.col(colSettings).FindOne(ctx,
		s.f(bson.E{Key: "key", Value: key}), &st)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			// خطای واقعی DB (قطعی، timeout، ...) — نباید بی‌صدا مثل «کلید نیست» رفتار کند.
			s.logErr("GetSetting("+key+")", err)
		}
		// مقدار خالی هم کش می‌شود تا در قطعی موقت DB، هر بار دوباره تلاش نکنیم
		// روی همان کلید (کش با SetSetting بعدی به‌روزرسانی می‌شود).
		s.settingsCacheSet(key, "")
		return ""
	}
	s.settingsCacheSet(key, st.Value)
	return st.Value
}

// SetSetting یک تنظیم را upsert می‌کند (اینترفیس upsert ندارد، دستی emulate می‌شود)
// و بلافاصله کش درون‌حافظه‌ای را به‌روز می‌کند تا GetSetting بعدی مقدار تازه ببیند.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	filter := s.f(bson.E{Key: "key", Value: key})
	var existing models.Setting
	err := s.col(colSettings).FindOne(ctx, filter, &existing)
	if errors.Is(err, mongo.ErrNoDocuments) {
		_, e := s.col(colSettings).InsertOne(ctx, &models.Setting{
			InstanceID: s.instanceID,
			Key:        key,
			Value:      value,
		})
		if e != nil {
			return e
		}
		s.settingsCacheSet(key, value)
		return nil
	}
	if err != nil {
		return err
	}
	if e := s.col(colSettings).UpdateOne(ctx, filter,
		bson.D{{Key: "$set", Value: bson.D{{Key: "value", Value: value}}}}); e != nil {
		return e
	}
	s.settingsCacheSet(key, value)
	return nil
}

// GetAllSettings همه‌ی تنظیمات این instance را به‌صورت map برمی‌گرداند و در
// همان حین کش درون‌حافظه‌ای را هم گرم می‌کند (پنل تنظیمات معمولاً اولین جایی
// است که همه‌ی کلیدها یک‌جا لازم می‌شوند).
func (s *Store) GetAllSettings(ctx context.Context) map[string]string {
	var list []models.Setting
	if err := s.col(colSettings).Find(ctx, s.f(), &list); err != nil {
		s.logErr("GetAllSettings", err)
		return map[string]string{}
	}
	m := make(map[string]string, len(list))
	for _, st := range list {
		m[st.Key] = st.Value
		s.settingsCacheSet(st.Key, st.Value)
	}
	return m
}

// ── کش درون‌حافظه‌ایِ تنظیمات ───────────────────────────────────

func (s *Store) settingsCacheGet(key string) (string, bool) {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	v, ok := s.settingsCache[key]
	return v, ok
}

func (s *Store) settingsCacheSet(key, value string) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	if s.settingsCache == nil {
		s.settingsCache = make(map[string]string)
	}
	s.settingsCache[key] = value
}

package store

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// الفبای کدها: حروف بزرگ، کوچک و عدد (بدون کاراکترهای مبهم 0/O/l/I حذف نشده‌اند
// چون کاربر صریحاً همه را خواست).
const codeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// الفبای پیشوند ربات: فقط حروف کوچک (مثل ux، kj).
const prefixAlphabet = "abcdefghijklmnopqrstuvwxyz"

// randString یک رشته‌ی تصادفی امن از روی alphabet می‌سازد.
func randString(alphabet string, n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		idx, err := crand.Int(crand.Reader, max)
		if err != nil {
			// fallback بسیار نادر — از زمان استفاده می‌کنیم
			b[i] = alphabet[time.Now().UnixNano()%int64(len(alphabet))]
			continue
		}
		b[i] = alphabet[idx.Int64()]
	}
	return string(b)
}

func (s *Store) CreateCode(ctx context.Context, c *models.Code) error {
	if c.ID == "" {
		c.ID = newID()
	}
	c.InstanceID = s.instanceID
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	if c.FileIDs == nil {
		c.FileIDs = []string{}
	}
	_, err := s.col(colCodes).InsertOne(ctx, c)
	return err
}

func (s *Store) FindCode(ctx context.Context, code string) (*models.Code, error) {
	var c models.Code
	err := s.col(colCodes).FindOne(ctx, s.f(bson.E{Key: "code", Value: code}), &c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) FindCodeByID(ctx context.Context, id string) (*models.Code, error) {
	var c models.Code
	err := s.col(colCodes).FindOne(ctx, s.f(bson.E{Key: "_id", Value: id}), &c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) UpdateCode(ctx context.Context, c *models.Code) error {
	c.UpdatedAt = time.Now()
	s.InvalidateCode(ctx, c.Code) // کش قدیمی باطل شود
	return s.col(colCodes).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: c.ID}),
		set(bson.D{
			{Key: "code", Value: c.Code},
			{Key: "type", Value: c.Type},
			{Key: "folder_id", Value: c.FolderID},
			{Key: "caption", Value: c.Caption},
			{Key: "thumbnail", Value: c.Thumbnail},
			{Key: "is_album", Value: c.IsAlbum},
			{Key: "max_use", Value: c.MaxUse},
			{Key: "used_count", Value: c.UsedCount},
			{Key: "expires_at", Value: c.ExpiresAt},
			{Key: "forward_lock", Value: c.ForwardLock},
			{Key: "anti_filter", Value: c.AntiFilter},
			{Key: "channel_lock", Value: c.ChannelLock},
			{Key: "auto_delete", Value: c.AutoDelete},
			{Key: "password", Value: c.Password},
			{Key: "download_limit", Value: c.DownloadLimit},
			{Key: "fake_likes", Value: c.FakeLikes},
			{Key: "fake_downloads", Value: c.FakeDownloads},
			{Key: "fake_views", Value: c.FakeViews},
			{Key: "force_seen", Value: c.ForceSeen},
			{Key: "force_react", Value: c.ForceReact},
			{Key: "sub_required", Value: c.SubRequired},
			{Key: "pending", Value: c.Pending},
			{Key: "file_ids", Value: c.FileIDs},
			{Key: "media_types", Value: c.MediaTypes},
		}))
}

func (s *Store) DeleteCode(ctx context.Context, id string) error {
	if c, err := s.FindCodeByID(ctx, id); err != nil {
		s.logErr("DeleteCode: lookup", err)
	} else if c != nil {
		s.InvalidateCode(ctx, c.Code)
	}
	return s.col(colCodes).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: id}))
}

// ── کش تحویل کد (کاهش درخواست به دیتابیس) ─────────────────────

type codeBundle struct {
	Code  models.Code   `json:"c"`
	Files []models.File `json:"f"`
}

func (s *Store) codeCacheKey(code string) string {
	return "upl:cc:" + s.instanceID + ":" + code
}

// InvalidateCode کش یک کد را پاک می‌کند.
func (s *Store) InvalidateCode(ctx context.Context, code string) {
	if s.cache != nil && code != "" {
		s.logErr("InvalidateCode", s.cache.Del(ctx, s.codeCacheKey(code)))
	}
}

// FindCodeForDelivery کد و فایل‌هایش را برای تحویل برمی‌گرداند و از کش Redis
// استفاده می‌کند تا درخواست‌های تکراری به دیتابیس کم شود. فقط کدهای «نامحدود»
// کش می‌شوند (برای کدهای محدود/یک‌بار، شمارش دقیق لازم است).
func (s *Store) FindCodeForDelivery(ctx context.Context, codeStr string) (*models.Code, []models.File, error) {
	if s.cache != nil {
		raw, err := s.cache.Get(ctx, s.codeCacheKey(codeStr))
		if err != nil {
			s.logErr("FindCodeForDelivery: cache get", err)
		}
		if raw != "" {
			var b codeBundle
			if json.Unmarshal([]byte(raw), &b) == nil {
				cc := b.Code
				return &cc, b.Files, nil
			}
		}
	}
	c, err := s.FindCode(ctx, codeStr)
	if err != nil || c == nil {
		return c, nil, err
	}
	files, err := s.GetFilesForCode(ctx, c.ID)
	if err != nil {
		return c, nil, err
	}
	if s.cache != nil && c.Type == models.CodeUnlimited {
		if data, e := json.Marshal(codeBundle{Code: *c, Files: files}); e != nil {
			s.logErr("FindCodeForDelivery: marshal", e)
		} else {
			s.logErr("FindCodeForDelivery: cache set", s.cache.Set(ctx, s.codeCacheKey(codeStr), string(data), 10*time.Minute))
		}
	}
	return c, files, nil
}

func (s *Store) IncrementCodeUse(ctx context.Context, id string) error {
	return s.col(colCodes).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "used_count", Value: 1}}},
			{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now()}}},
		})
}

func (s *Store) ListCodes(ctx context.Context, folderID string, page, limit int) ([]models.Code, int64, error) {
	var extra []bson.E
	if folderID != "" && folderID != "root" {
		extra = append(extra, bson.E{Key: "folder_id", Value: folderID})
	}
	filter := s.f(extra...)
	total, err := s.col(colCodes).CountDocuments(ctx, filter)
	s.logErr("ListCodes: count", err)
	if page < 1 {
		page = 1
	}
	var codes []models.Code
	err = s.col(colCodes).Find(ctx, filter, &codes,
		ports_sortDesc("created_at"),
		ports_skip(int64((page-1)*limit)),
		ports_limit(int64(limit)),
	)
	return codes, total, err
}

// escapeMongoRegex متاکاراکترهای regex را escape می‌کند تا ورودی کاربر
// به‌صورت یک رشته‌ی لغوی (literal) در primitive.Regex استفاده شود و از
// حملات ReDoS جلوگیری شود.
func escapeMongoRegex(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '.', '*', '+', '?', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (s *Store) SearchCodes(ctx context.Context, query string) ([]models.Code, error) {
	rx := primitive.Regex{Pattern: escapeMongoRegex(query), Options: "i"}
	or := bson.A{
		bson.D{{Key: "code", Value: rx}},
		bson.D{{Key: "caption", Value: rx}},
	}
	// جستجو بر اساس نوع رسانه (مثل «ویدیو»، «عکس»)
	if t := mediaTypeKeyword(query); t != "" {
		or = append(or, bson.D{{Key: "media_types", Value: t}})
	}
	filter := s.f(bson.E{Key: "$or", Value: or})
	var codes []models.Code
	err := s.col(colCodes).Find(ctx, filter, &codes, ports_limit(20))
	return codes, err
}

// mediaTypeKeyword عبارت کاربر را به نوع رسانه نگاشت می‌کند (فارسی/انگلیسی).
func mediaTypeKeyword(q string) string {
	switch {
	case containsAny(q, "عکس", "تصویر", "photo", "image"):
		return "photo"
	case containsAny(q, "ویدیو", "فیلم", "video"):
		return "video"
	case containsAny(q, "صوت", "آهنگ", "اهنگ", "موزیک", "audio", "music"):
		return "audio"
	case containsAny(q, "گیف", "gif", "animation"):
		return "animation"
	case containsAny(q, "ویس", "voice"):
		return "voice"
	case containsAny(q, "استیکر", "sticker"):
		return "sticker"
	case containsAny(q, "فایل", "سند", "document", "file"):
		return "document"
	}
	return ""
}

func containsAny(s string, subs ...string) bool {
	s = strings.ToLower(s)
	for _, sub := range subs {
		if strings.Contains(s, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// RefreshCodeTypes انواع رسانه‌ی یک کد را از روی فایل‌هایش به‌روز می‌کند.
func (s *Store) RefreshCodeTypes(ctx context.Context, codeID string) {
	files, err := s.GetFilesForCode(ctx, codeID)
	if err != nil || len(files) == 0 {
		return
	}
	seen := map[string]bool{}
	var types []string
	for _, f := range files {
		if !seen[f.FileType] {
			seen[f.FileType] = true
			types = append(types, f.FileType)
		}
	}
	s.logErr("RefreshCodeTypes", s.col(colCodes).UpdateOne(ctx, s.f(bson.E{Key: "_id", Value: codeID}),
		set(bson.D{{Key: "media_types", Value: types}})))
}

// ListCodesSorted کدها را بر اساس یک فیلد (نزولی) مرتب و برمی‌گرداند.
// برای منوهای کاربری: created_at (جدیدترین)، fake_views (پربازدید)، used_count (محبوب).
func (s *Store) ListCodesSorted(ctx context.Context, field string, limit int) ([]models.Code, error) {
	var codes []models.Code
	err := s.col(colCodes).Find(ctx,
		s.f(bson.E{Key: "pending", Value: bson.D{{Key: "$ne", Value: true}}}), &codes,
		ports_sortDesc(field), ports_limit(int64(limit)))
	return codes, err
}

// ListPendingCodes کدهای در انتظار تایید را برمی‌گرداند.
func (s *Store) ListPendingCodes(ctx context.Context, limit int) ([]models.Code, error) {
	var codes []models.Code
	err := s.col(colCodes).Find(ctx,
		s.f(bson.E{Key: "pending", Value: true}), &codes,
		ports_sortAsc("created_at"), ports_limit(int64(limit)))
	return codes, err
}

// ApproveCode یک کد را از حالت انتظار خارج می‌کند.
func (s *Store) ApproveCode(ctx context.Context, id string) error {
	if c, err := s.FindCodeByID(ctx, id); err != nil {
		s.logErr("ApproveCode: lookup", err)
	} else if c != nil {
		s.InvalidateCode(ctx, c.Code)
	}
	return s.col(colCodes).UpdateOne(ctx, s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "pending", Value: false}}))
}

func (s *Store) CodeExists(ctx context.Context, code string) bool {
	n, err := s.col(colCodes).CountDocuments(ctx, s.f(bson.E{Key: "code", Value: code}))
	s.logErr("CodeExists", err)
	return n > 0
}

// GenerateUniqueCode یک کد یکتای الفبا-عددی با پیشوند اختصاصی ربات می‌سازد،
// مثل «ux_Ab3xK9pQ». اگر پیشوند ست نشده باشد، فقط بدنه ساخته می‌شود.
func (s *Store) GenerateUniqueCode(ctx context.Context) string {
	prefix := s.GetSetting(ctx, models.SettingCodePrefix)
	for {
		code := randString(codeAlphabet, 8)
		if prefix != "" {
			code = prefix + "_" + code
		}
		if !s.CodeExists(ctx, code) {
			return code
		}
	}
}

// EnsureCodePrefix پیشوند اختصاصی ربات را یک‌بار می‌سازد و ذخیره می‌کند.
// اگر از قبل وجود داشته باشد، همان را برمی‌گرداند (تغییر نمی‌کند).
func (s *Store) EnsureCodePrefix(ctx context.Context) string {
	if p := s.GetSetting(ctx, models.SettingCodePrefix); p != "" {
		return p
	}
	p := randString(prefixAlphabet, 2)
	s.logErr("EnsureCodePrefix", s.SetSetting(ctx, models.SettingCodePrefix, p))
	return p
}

package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *Store { return &Store{db: db} }

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&RegisteredImage{}, &AllowedCaller{})
}

// ── RegisteredImage ────────────────────────────────────────

func (s *Store) CreateImage(ctx context.Context, img *RegisteredImage) error {
	return s.db.WithContext(ctx).Create(img).Error
}

func (s *Store) ListImages(ctx context.Context) ([]RegisteredImage, error) {
	var list []RegisteredImage
	return list, s.db.WithContext(ctx).Order("name, tag").Find(&list).Error
}

// FindImage یک image فعال با name+tag دقیق را برمی‌گرداند (nil یعنی یا
// وجود ندارد یا غیرفعال است — عمداً یک نتیجه، تا فراخوان مجبور به یک چک
// "== nil" ساده برای رد کردن باشد، نه بررسی جداگانه‌ی IsActive).
func (s *Store) FindImage(ctx context.Context, name, tag string) (*RegisteredImage, error) {
	var img RegisteredImage
	err := s.db.WithContext(ctx).
		Where("name = ? AND tag = ? AND is_active = true", name, tag).
		First(&img).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &img, err
}

func (s *Store) DeleteImage(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&RegisteredImage{}, "id = ?", id).Error
}

func (s *Store) SetImageActive(ctx context.Context, id uuid.UUID, active bool) error {
	return s.db.WithContext(ctx).Model(&RegisteredImage{}).
		Where("id = ?", id).Update("is_active", active).Error
}

// FindImageByID برای مسیرهایی که فقط id دارند (آپلود/دانلود فایل) — برخلاف
// FindImage عمداً غیرفعال‌ها را هم برمی‌گرداند، چون آپلود فایل روی یک image
// موقتاً غیرفعال هم باید ممکن باشد.
func (s *Store) FindImageByID(ctx context.Context, id uuid.UUID) (*RegisteredImage, error) {
	var img RegisteredImage
	err := s.db.WithContext(ctx).First(&img, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &img, err
}

// SetImageFile متادیتای فایل آپلودشده (مسیر/چک‌سام/اندازه) را بعد از ذخیره‌ی
// موفق آن روی دیسک ثبت می‌کند.
func (s *Store) SetImageFile(ctx context.Context, id uuid.UUID, path, sha256 string, size int64) error {
	return s.db.WithContext(ctx).Model(&RegisteredImage{}).
		Where("id = ?", id).
		Updates(map[string]any{"file_path": path, "file_sha256": sha256, "file_size": size}).Error
}

// ── AllowedCaller ──────────────────────────────────────────

func (s *Store) CreateCaller(ctx context.Context, c *AllowedCaller) error {
	return s.db.WithContext(ctx).Create(c).Error
}

func (s *Store) ListCallers(ctx context.Context) ([]AllowedCaller, error) {
	var list []AllowedCaller
	return list, s.db.WithContext(ctx).Order("created_at").Find(&list).Error
}

// ListActiveCallers فقط callerهای فعال — روی هر درخواست صدا زده می‌شود، پس
// این کوئری باید کوچک/سریع بماند؛ برای تعداد سرورهای این پلتفرم (ده‌ها، نه
// میلیون‌ها) مشکلی نیست. اگر تعداد خیلی زیاد شد، یک کش کوتاه‌مدت اضافه کنید.
func (s *Store) ListActiveCallers(ctx context.Context) ([]AllowedCaller, error) {
	var list []AllowedCaller
	return list, s.db.WithContext(ctx).Where("is_active = true").Find(&list).Error
}

func (s *Store) DeleteCaller(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&AllowedCaller{}, "id = ?", id).Error
}

func (s *Store) SetCallerActive(ctx context.Context, id uuid.UUID, active bool) error {
	return s.db.WithContext(ctx).Model(&AllowedCaller{}).
		Where("id = ?", id).Update("is_active", active).Error
}

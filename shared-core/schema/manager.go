// Package schema مدیریت PostgreSQL schema های اختصاصی هر bot instance را انجام می‌دهد.
//
// به‌جای ایجاد یک دیتابیس جدید برای هر ربات، از PostgreSQL schemas استفاده می‌شود:
//
//	botmanager DB
//	├── public          ← botmanager/apimanager
//	├── inst_abc12345   ← uploader-bot instance
//	└── inst_def67890   ← vpn-bot instance
//
// هر instance با یک search_path جداگانه به DB وصل می‌شود:
//
//	DSN: postgres://...?search_path=inst_abc12345
package schema

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Create یک schema جدید در دیتابیس می‌سازد.
// idempotent: اگه schema قبلاً وجود داشت، خطا نمی‌دهد.
func Create(ctx context.Context, db *gorm.DB, schemaName string) error {
	if err := validate(schemaName); err != nil {
		return err
	}
	// CREATE SCHEMA IF NOT EXISTS — ایمن در برابر race condition
	return db.WithContext(ctx).Exec(
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName),
	).Error
}

// Drop یک schema و همه جداول آن را حذف می‌کند.
// از DROP CASCADE استفاده می‌شود — با احتیاط استفاده کنید.
func Drop(ctx context.Context, db *gorm.DB, schemaName string) error {
	if err := validate(schemaName); err != nil {
		return err
	}
	return db.WithContext(ctx).Exec(
		fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName),
	).Error
}

// DSN یک DSN جدید با search_path برای schema مشخص می‌سازد.
// ربات‌های deploy شده با این DSN به DB وصل می‌شوند.
func DSN(baseDSN, schemaName string) (string, error) {
	if err := validate(schemaName); err != nil {
		return "", err
	}
	// اگه query string داره
	if strings.Contains(baseDSN, "?") {
		return baseDSN + "&search_path=" + schemaName, nil
	}
	return baseDSN + "?search_path=" + schemaName, nil
}

// Exists بررسی می‌کند schema وجود دارد یا نه.
func Exists(ctx context.Context, db *gorm.DB, schemaName string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = ?",
		schemaName,
	).Scan(&count).Error
	return count > 0, err
}

// ListInstanceSchemas همه schema های instance را برمی‌گرداند.
func ListInstanceSchemas(ctx context.Context, db *gorm.DB) ([]string, error) {
	var schemas []string
	err := db.WithContext(ctx).Raw(
		"SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'inst_%'",
	).Scan(&schemas).Error
	return schemas, err
}

// validate نام schema را بررسی می‌کند — فقط حروف، اعداد و underscore مجاز است.
func validate(name string) error {
	if name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("invalid schema name %q: only lowercase letters, digits, and underscores allowed", name)
		}
	}
	return nil
}

// Package migrate موتور migration ورژن‌دار پلتفرم.
//
// هر سرویس یک پوشه در migrations/ دارد با فایل‌های <NNNN>_<name>.sql.
// وضعیت اعمال‌شده در جدول schema_migrations داخل دیتابیس خودِ همان سرویس
// نگه داشته می‌شود (نسخه + checksum) — پس هر دیتابیس خودش می‌داند روی چه
// نسخه‌ای است و دستکاری فایل‌های اعمال‌شده هم تشخیص داده می‌شود.
package migrate

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Migration یک فایل SQL ورژن‌دار.
type Migration struct {
	Version  int64
	Name     string // بخش بعد از عدد در اسم فایل، بدون .sql
	FileName string
	SQL      string
	Checksum string // sha256 محتوا
}

// AppliedRow یک ردیف از جدول schema_migrations.
type AppliedRow struct {
	Version  int64
	Name     string
	Checksum string
}

var fileRe = regexp.MustCompile(`^(\d+)_([A-Za-z0-9_\-]+)\.sql$`)

// Load همه‌ی migrationهای یک سرویس را از fsys (پوشه‌ی <service>/) می‌خواند.
func Load(fsys fs.FS, service string) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, service)
	if err != nil {
		return nil, fmt.Errorf("migrationهای سرویس %s پیدا نشد: %w", service, err)
	}
	var out []Migration
	seen := map[int64]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		m := fileRe.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("%s/%s: اسم فایل باید فرمت <NNNN>_<name>.sql داشته باشد", service, e.Name())
		}
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("%s/%s: نسخه‌ی نامعتبر", service, e.Name())
		}
		if prev, dup := seen[v]; dup {
			return nil, fmt.Errorf("%s: نسخه‌ی %d دوبار تعریف شده (%s و %s)", service, v, prev, e.Name())
		}
		seen[v] = e.Name()
		b, err := fs.ReadFile(fsys, service+"/"+e.Name())
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		out = append(out, Migration{
			Version:  v,
			Name:     m[2],
			FileName: e.Name(),
			SQL:      string(b),
			Checksum: hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	if len(out) == 0 {
		return nil, fmt.Errorf("سرویس %s هیچ فایل migration ندارد", service)
	}
	return out, nil
}

// ── اتصال ─────────────────────────────────────────────────

// ServiceDSN دیتابیسِ داخل DSN سرور را با دیتابیس سرویس عوض می‌کند.
// DSN باید فرمت URL داشته باشد (postgres://user:pass@host:port/db?...) —
// همان فرمتی که همه‌ی .env های پروژه استفاده می‌کنند.
func ServiceDSN(serverDSN, dbName string) (string, error) {
	u, err := url.Parse(serverDSN)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return "", fmt.Errorf("DSN نامعتبر (فرمت postgres://... لازم است): %q", serverDSN)
	}
	u.Path = "/" + dbName
	return u.String(), nil
}

// Connect به یک DSN وصل شده و ping می‌کند.
func Connect(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// EnsureDatabase اگر دیتابیس سرویس وجود نداشته باشد، از طریق اتصالِ ادمین
// (همان DSN ورودی، بدون تغییر) آن را می‌سازد.
func EnsureDatabase(ctx context.Context, serverDSN, dbName string) (created bool, err error) {
	admin, err := Connect(ctx, serverDSN)
	if err != nil {
		return false, fmt.Errorf("اتصال ادمین: %w", err)
	}
	defer admin.Close()

	var exists bool
	if err := admin.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists); err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	// CREATE DATABASE پارامتر bind نمی‌پذیرد؛ dbName از Registry ثابت خودمان
	// می‌آید نه ورودی آزاد کاربر، ولی برای اطمینان quote می‌کنیم.
	if _, err := admin.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %q`, dbName)); err != nil {
		return false, fmt.Errorf("CREATE DATABASE %s: %w", dbName, err)
	}
	return true, nil
}

// ── جدول وضعیت ────────────────────────────────────────────

const ensureTableSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version    BIGINT PRIMARY KEY,
	name       TEXT        NOT NULL,
	checksum   TEXT        NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

// Applied لیست نسخه‌های اعمال‌شده را برمی‌گرداند (جدول را هم اگر نبود می‌سازد).
func Applied(ctx context.Context, db *sql.DB) ([]AppliedRow, error) {
	if _, err := db.ExecContext(ctx, ensureTableSQL); err != nil {
		return nil, fmt.Errorf("ساخت جدول schema_migrations: %w", err)
	}
	rows, err := db.QueryContext(ctx, "SELECT version, name, checksum FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AppliedRow
	for rows.Next() {
		var r AppliedRow
		if err := rows.Scan(&r.Version, &r.Name, &r.Checksum); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Verify چک می‌کند migrationهای اعمال‌شده با فایل‌های فعلی یکی باشند.
// checksum متفاوت یعنی کسی فایل اعمال‌شده را عوض کرده — خطای سخت، چون
// دیگر معلوم نیست دیتابیس واقعاً چه schema ای دارد.
func Verify(applied []AppliedRow, files []Migration) error {
	byVer := map[int64]Migration{}
	for _, f := range files {
		byVer[f.Version] = f
	}
	for _, a := range applied {
		f, ok := byVer[a.Version]
		if !ok {
			return fmt.Errorf("نسخه‌ی %d (%s) در دیتابیس اعمال شده ولی فایلش دیگر وجود ندارد", a.Version, a.Name)
		}
		if f.Checksum != a.Checksum {
			return fmt.Errorf("checksum نسخه‌ی %d (%s) با فایل فعلی فرق دارد — فایل اعمال‌شده را عوض نکنید؛ نسخه‌ی جدید بسازید", a.Version, f.FileName)
		}
	}
	return nil
}

// ── اعمال ─────────────────────────────────────────────────

// Up همه‌ی migrationهای اعمال‌نشده تا نسخه‌ی target را اجرا می‌کند.
// target=0 یعنی تا آخرین نسخه. هر migration در یک تراکنش جدا اجرا و ثبت
// می‌شود، پس شکست وسط راه نسخه‌های قبلی را خراب نمی‌کند.
func Up(ctx context.Context, db *sql.DB, files []Migration, target int64) (appliedNow []Migration, err error) {
	applied, err := Applied(ctx, db)
	if err != nil {
		return nil, err
	}
	if err := Verify(applied, files); err != nil {
		return nil, err
	}
	var current int64
	done := map[int64]bool{}
	for _, a := range applied {
		done[a.Version] = true
		if a.Version > current {
			current = a.Version
		}
	}
	if target == 0 {
		target = files[len(files)-1].Version
	}
	if target < current {
		return nil, fmt.Errorf("دیتابیس روی نسخه‌ی %d است؛ برگشت به %d پشتیبانی نمی‌شود (down migration نداریم)", current, target)
	}
	for _, f := range files {
		if f.Version > target || done[f.Version] {
			continue
		}
		if err := applyOne(ctx, db, f); err != nil {
			return appliedNow, fmt.Errorf("نسخه‌ی %d (%s): %w", f.Version, f.FileName, err)
		}
		appliedNow = append(appliedNow, f)
	}
	return appliedNow, nil
}

func applyOne(ctx context.Context, db *sql.DB, m Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)",
		m.Version, m.Name, m.Checksum); err != nil {
		return err
	}
	return tx.Commit()
}

// Mark نسخه‌ها را تا target بدون اجرای SQL به‌عنوان اعمال‌شده ثبت می‌کند.
// برای دیتابیس‌هایی که schema شان از قبل (مثلاً با AutoMigrate خود سرویس)
// ساخته شده و فقط باید به سیستم ورژن‌دار وارد شوند.
func Mark(ctx context.Context, db *sql.DB, files []Migration, target int64) (marked []Migration, err error) {
	applied, err := Applied(ctx, db)
	if err != nil {
		return nil, err
	}
	done := map[int64]bool{}
	for _, a := range applied {
		done[a.Version] = true
	}
	if target == 0 {
		target = files[len(files)-1].Version
	}
	for _, f := range files {
		if f.Version > target || done[f.Version] {
			continue
		}
		if _, err := db.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)",
			f.Version, f.Name, f.Checksum); err != nil {
			return marked, err
		}
		marked = append(marked, f)
	}
	return marked, nil
}

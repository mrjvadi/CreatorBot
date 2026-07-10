// dbmigrate — سرویس متمرکز migration ورژن‌دار دیتابیس‌های پلتفرم.
//
// هر سرویس Postgres دار یک پوشه در migrations/ دارد؛ نسخه‌ی ۱ (baseline)
// از AutoMigrate واقعی خود سرویس تولید شده و نسخه‌های بعدی را توسعه‌دهنده
// با `dbmigrate new` اضافه می‌کند. وضعیت هر دیتابیس در جدول
// schema_migrations خودش نگه داشته می‌شود.
//
// مثال‌ها:
//
//	dbmigrate list                                # سرویس‌ها و نسخه‌ها
//	dbmigrate status -service all                 # وضعیت همه
//	dbmigrate up -service botpay                  # تا آخرین نسخه
//	dbmigrate up -service ads-bot -version 2      # تا نسخه‌ی مشخص
//	dbmigrate mark -service botmanager -version 1 # ثبت بدون اجرا (DB موجود)
//	dbmigrate new -service botpay -name add_x     # ساخت فایل نسخه‌ی بعدی
//
// DSN از فلگ -dsn یا env POSTGRES_DSN خوانده می‌شود؛ اسم دیتابیس داخل آن
// مهم نیست — برای هر سرویس به‌صورت خودکار با دیتابیس خودش عوض می‌شود.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mrjvadi/creatorbot/dbmigrate/internal/migrate"
	"github.com/mrjvadi/creatorbot/dbmigrate/migrations"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	service := fs.String("service", "", "اسم سرویس (یا all برای up/status)")
	version := fs.Int64("version", 0, "نسخه‌ی هدف (0 = آخرین)")
	dsn := fs.String("dsn", os.Getenv("POSTGRES_DSN"), "DSN سرور Postgres (پیش‌فرض: env POSTGRES_DSN)")
	noCreateDB := fs.Bool("no-create-db", false, "دیتابیس را اگر نبود نساز (پیش‌فرض: می‌سازد)")
	name := fs.String("name", "", "اسم migration جدید (فقط برای new)")
	fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var err error
	switch cmd {
	case "list":
		err = cmdList()
	case "status":
		err = cmdStatus(ctx, *dsn, *service)
	case "up":
		err = cmdUp(ctx, *dsn, *service, *version, !*noCreateDB)
	case "mark":
		err = cmdMark(ctx, *dsn, *service, *version)
	case "new":
		err = cmdNew(*service, *name)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "دستور ناشناخته: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "خطا:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`dbmigrate — migration ورژن‌دار دیتابیس‌های CreatorBot

دستورها:
  list                                 لیست سرویس‌ها و نسخه‌های موجود
  status -service <svc|all>            وضعیت نسخه‌ی هر دیتابیس
  up     -service <svc|all> [-version N]   اعمال migrationها تا نسخه‌ی هدف
  mark   -service <svc> [-version N]   ثبت نسخه بدون اجرای SQL (برای DB موجود)
  new    -service <svc> -name <slug>   ساخت فایل migration نسخه‌ی بعدی

فلگ‌ها:
  -dsn            DSN سرور Postgres (پیش‌فرض: env POSTGRES_DSN)
  -no-create-db   دیتابیس را اگر نبود نساز`)
}

// resolveServices "-service all" را به کل Registry باز می‌کند.
func resolveServices(arg string) ([]migrate.Service, error) {
	if arg == "" {
		return nil, fmt.Errorf("فلگ -service لازم است (یا -service all)")
	}
	if strings.EqualFold(arg, "all") {
		return migrate.Registry, nil
	}
	s, err := migrate.Find(arg)
	if err != nil {
		return nil, err
	}
	return []migrate.Service{s}, nil
}

func cmdList() error {
	fmt.Printf("%-20s %-15s %-8s %s\n", "SERVICE", "DATABASE", "LATEST", "NOTE")
	for _, s := range migrate.Registry {
		files, err := migrate.Load(migrations.FS, s.Name)
		if err != nil {
			return err
		}
		fmt.Printf("%-20s %-15s %-8d %s\n", s.Name, s.DBName, files[len(files)-1].Version, s.Note)
	}
	return nil
}

func cmdStatus(ctx context.Context, dsn, serviceArg string) error {
	if serviceArg == "" {
		serviceArg = "all"
	}
	svcs, err := resolveServices(serviceArg)
	if err != nil {
		return err
	}
	if dsn == "" {
		return fmt.Errorf("DSN تنظیم نشده (فلگ -dsn یا env POSTGRES_DSN)")
	}
	fmt.Printf("%-20s %-15s %-10s %-8s %s\n", "SERVICE", "DATABASE", "APPLIED", "LATEST", "PENDING")
	for _, s := range svcs {
		files, err := migrate.Load(migrations.FS, s.Name)
		if err != nil {
			return err
		}
		latest := files[len(files)-1].Version

		svcDSN, err := migrate.ServiceDSN(dsn, s.DBName)
		if err != nil {
			return err
		}
		db, err := migrate.Connect(ctx, svcDSN)
		if err != nil {
			fmt.Printf("%-20s %-15s %-10s %-8d (دیتابیس در دسترس نیست)\n", s.Name, s.DBName, "-", latest)
			continue
		}
		applied, err := migrate.Applied(ctx, db)
		db.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
		var current int64
		for _, a := range applied {
			if a.Version > current {
				current = a.Version
			}
		}
		pending := 0
		for _, f := range files {
			if f.Version > current {
				pending++
			}
		}
		fmt.Printf("%-20s %-15s %-10d %-8d %d\n", s.Name, s.DBName, current, latest, pending)
	}
	return nil
}

func cmdUp(ctx context.Context, dsn, serviceArg string, target int64, createDB bool) error {
	svcs, err := resolveServices(serviceArg)
	if err != nil {
		return err
	}
	if dsn == "" {
		return fmt.Errorf("DSN تنظیم نشده (فلگ -dsn یا env POSTGRES_DSN)")
	}
	if len(svcs) > 1 && target != 0 {
		return fmt.Errorf("فلگ -version با -service all معنی ندارد — سرویس مشخص بدهید")
	}
	for _, s := range svcs {
		files, err := migrate.Load(migrations.FS, s.Name)
		if err != nil {
			return err
		}
		if createDB {
			created, err := migrate.EnsureDatabase(ctx, dsn, s.DBName)
			if err != nil {
				return fmt.Errorf("%s: %w", s.Name, err)
			}
			if created {
				fmt.Printf("%s: دیتابیس %s ساخته شد\n", s.Name, s.DBName)
			}
		}
		svcDSN, err := migrate.ServiceDSN(dsn, s.DBName)
		if err != nil {
			return err
		}
		db, err := migrate.Connect(ctx, svcDSN)
		if err != nil {
			return fmt.Errorf("%s: اتصال به %s: %w", s.Name, s.DBName, err)
		}
		applied, err := migrate.Up(ctx, db, files, target)
		db.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
		if len(applied) == 0 {
			fmt.Printf("%s: به‌روز است\n", s.Name)
			continue
		}
		for _, m := range applied {
			fmt.Printf("%s: نسخه‌ی %d (%s) اعمال شد\n", s.Name, m.Version, m.Name)
		}
	}
	return nil
}

func cmdMark(ctx context.Context, dsn, serviceArg string, target int64) error {
	if strings.EqualFold(serviceArg, "all") {
		return fmt.Errorf("mark فقط برای یک سرویس مشخص — این دستور یعنی «schema از قبل هست»، که باید آگاهانه برای هر سرویس تصمیم بگیرید")
	}
	svcs, err := resolveServices(serviceArg)
	if err != nil {
		return err
	}
	if dsn == "" {
		return fmt.Errorf("DSN تنظیم نشده (فلگ -dsn یا env POSTGRES_DSN)")
	}
	s := svcs[0]
	files, err := migrate.Load(migrations.FS, s.Name)
	if err != nil {
		return err
	}
	svcDSN, err := migrate.ServiceDSN(dsn, s.DBName)
	if err != nil {
		return err
	}
	db, err := migrate.Connect(ctx, svcDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	marked, err := migrate.Mark(ctx, db, files, target)
	if err != nil {
		return err
	}
	if len(marked) == 0 {
		fmt.Printf("%s: چیزی برای ثبت نبود\n", s.Name)
		return nil
	}
	for _, m := range marked {
		fmt.Printf("%s: نسخه‌ی %d (%s) بدون اجرا ثبت شد\n", s.Name, m.Version, m.Name)
	}
	return nil
}

// cmdNew فایل migration نسخه‌ی بعدی را در سورس (نه embed) می‌سازد.
func cmdNew(serviceArg, name string) error {
	if name == "" {
		return fmt.Errorf("فلگ -name لازم است (مثلاً -name add_refund_column)")
	}
	if !fileSlugRe.MatchString(name) {
		return fmt.Errorf("اسم فقط حروف/عدد/underscore: %q", name)
	}
	svcs, err := resolveServices(serviceArg)
	if err != nil {
		return err
	}
	if len(svcs) != 1 {
		return fmt.Errorf("new فقط برای یک سرویس مشخص")
	}
	s := svcs[0]

	// پوشه‌ی سورس migrations را پیدا کن — بسته به این‌که از کجا اجرا شده.
	dir := ""
	for _, cand := range []string{
		filepath.Join("migrations", s.Name),
		filepath.Join("dbmigrate", "migrations", s.Name),
	} {
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			dir = cand
			break
		}
	}
	if dir == "" {
		return fmt.Errorf("پوشه‌ی migrations/%s پیدا نشد — از ریشه‌ی repo یا پوشه‌ی dbmigrate اجرا کنید", s.Name)
	}

	files, err := migrate.Load(os.DirFS(filepath.Dir(dir)), s.Name)
	if err != nil {
		return err
	}
	next := files[len(files)-1].Version + 1
	path := filepath.Join(dir, fmt.Sprintf("%04d_%s.sql", next, name))
	body := fmt.Sprintf(`-- %04d_%s — سرویس %s (دیتابیس: %s)
--
-- TODO: SQL این نسخه را این‌جا بنویسید. کل فایل در یک تراکنش اجرا می‌شود.
-- بعد از نوشتن، باینری dbmigrate را دوباره build کنید (فایل‌ها embed هستند)
-- و با این دستور اعمال کنید:
--   dbmigrate up -service %s -version %d
`, next, name, s.Name, s.DBName, s.Name, next)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	fmt.Println("ساخته شد:", path)
	return nil
}

var fileSlugRe = regexp.MustCompile(`^[a-z0-9_]+$`)

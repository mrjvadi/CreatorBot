// Package ipallow کنترل دسترسی بر اساس IP/CIDR (و اختیاراً دامنه) را پیاده
// می‌کند — دقیقاً همان چیزی که هدف این سرویس بود: whitelist بر اساس IP، نه
// روی HTTP API key به‌تنهایی.
//
// چرا این روی HTTP است، نه NATS (بر خلاف بقیه‌ی پلتفرم): NATS یک message
// bus است — پیام‌رسانی‌اش IP واقعی فرستنده را در سطح اپلیکیشن در اختیار
// responder نمی‌گذارد (فقط با یک ادعای داخل payload که به‌راحتی جعل‌پذیر
// است). یک اتصال HTTP مستقیم، برعکس، IP واقعی TCP را در سطح شبکه به سرور
// می‌دهد — این تنها راهی است که «بر اساس IP» واقعاً به‌معنای امنیتی درست
// (نه فقط یک ادعا) اجرا شود. به همین دلیل image-registry تنها سرویس این
// پلتفرم است که با HTTP ساده کار می‌کند، نه NATS.
package ipallow

import (
	"context"
	"net"

	"github.com/mrjvadi/creatorbot/image-registry/internal/store"
)

// Checker تصمیم می‌گیرد یک IP اجازه‌ی دسترسی (و چه سطحی از دسترسی) دارد.
type Checker struct {
	st *store.Store
}

func New(st *store.Store) *Checker {
	return &Checker{st: st}
}

// Result نتیجه‌ی چک یک IP.
type Result struct {
	Allowed  bool
	CanWrite bool
	Label    string
}

// Check همه‌ی callerهای فعال را می‌گیرد و IP ورودی را با CIDR هرکدام مقایسه
// می‌کند؛ اگر Domain هم روی آن caller ست شده باشد، دامنه resolve و IP باید
// در نتیجه‌ی آن هم باشد (لایه‌ی دوم، نه جایگزین CIDR — رجوع به کامنت
// AllowedCaller.Domain در models.go برای محدودیت‌های DNS).
//
// اگر چند caller مطابقت داشتند، هرکدام CanWrite=true داشت برنده می‌شود
// (یعنی یک IP که هم در یک ردیف read-only و هم در یک ردیف read-write هست،
// دسترسی نوشتن می‌گیرد — تصمیم عمدی برای سادگی، نه یک باگ).
func (c *Checker) Check(ctx context.Context, remoteIP string) (Result, error) {
	ip := net.ParseIP(remoteIP)
	if ip == nil {
		return Result{}, nil
	}

	callers, err := c.st.ListActiveCallers(ctx)
	if err != nil {
		return Result{}, err
	}

	res := Result{}
	for _, caller := range callers {
		_, ipNet, err := net.ParseCIDR(normalizeCIDR(caller.CIDR))
		if err != nil {
			continue // یک ردیف بد پیکربندی‌شده نباید کل چک را خراب کند
		}
		if !ipNet.Contains(ip) {
			continue
		}
		if caller.Domain != "" && !domainResolvesTo(caller.Domain, ip) {
			continue
		}
		res.Allowed = true
		res.Label = caller.Label
		if caller.CanWrite {
			res.CanWrite = true
		}
	}
	return res, nil
}

// normalizeCIDR اجازه می‌دهد کاربر یک IP تکی بدون "/32" هم وارد کند.
func normalizeCIDR(s string) string {
	if _, _, err := net.ParseCIDR(s); err == nil {
		return s
	}
	if ip := net.ParseIP(s); ip != nil {
		if ip.To4() != nil {
			return s + "/32"
		}
		return s + "/128"
	}
	return s
}

// domainResolvesTo — چک ساده‌ی DNS. محدودیت شناخته‌شده: DNS می‌تواند کش شود،
// دیر به‌روز شود، یا (در تئوری) مسموم شود؛ این‌جا فقط یک لایه‌ی دوم اضافه
// روی CIDR است، نه منبع اصلی اعتماد.
func domainResolvesTo(domain string, ip net.IP) bool {
	addrs, err := net.LookupIP(domain)
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if a.Equal(ip) {
			return true
		}
	}
	return false
}

// Package payment کلاینت درگاه‌های پرداخت آنلاین ایرانی (زرین‌پال، زیبال).
//
// الگوی استفاده در ربات:
//  1. Request: ساخت تراکنش و گرفتن لینک پرداخت → لینک برای کاربر فرستاده می‌شود.
//  2. کاربر پرداخت می‌کند و دکمه «پرداخت کردم» را می‌زند.
//  3. Verify: صحت پرداخت با authority/trackId بررسی و در صورت موفقیت اشتراک فعال می‌شود.
package payment

import (
	"net/http"
	"time"
)

// Gateway اینترفیس مشترک همه‌ی درگاه‌ها.
type Gateway interface {
	// Request یک تراکنش می‌سازد. amount به ریال است.
	// خروجی: شناسه‌ی پیگیری (authority/trackId) و لینک پرداخت.
	Request(amount int64, desc, callback string) (ref, payURL string, err error)

	// Verify صحت پرداخت را بررسی می‌کند. amount به ریال.
	Verify(ref string, amount int64) (paid bool, trackingCode string, err error)
}

// httpClient کلاینت مشترک با timeout.
var httpClient = &http.Client{Timeout: 20 * time.Second}

// New درگاه را بر اساس نام می‌سازد. اگر ناشناخته باشد nil برمی‌گرداند.
func New(name, merchant string) Gateway {
	switch name {
	case "zarinpal":
		return &Zarinpal{Merchant: merchant}
	case "zibal":
		return &Zibal{Merchant: merchant}
	}
	return nil
}

// TomanToRial تومان را به ریال تبدیل می‌کند (درگاه‌ها ریالی‌اند).
func TomanToRial(toman float64) int64 { return int64(toman * 10) }

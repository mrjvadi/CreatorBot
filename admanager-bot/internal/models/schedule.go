// schedule.go — محاسبات خالصِ زمان‌بندی (بدون وابستگی به Mongo/Telegram)،
// مشترک بین scheduler و لایه‌ی tgbot (نمای زمان‌بندی). چون این فایل فقط به
// "time" وابسته است، به‌راحتی و بدون نیاز به دیتابیس/شبکه قابل تست است —
// نگاه کنید به schedule_test.go و NEEDS.md (که نبودِ تست برای منطق چرخشی
// را به‌عنوان یک نقطه‌ضعف مشخص کرده بود).
package models

import "time"

// StartTimeOnDay ساعت/دقیقه‌ی داده‌شده را روی تاریخِ همان day می‌سازد.
func StartTimeOnDay(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}

// InDailyWindow مشخص می‌کند آیا لحظه‌ی now داخل بازه‌ی روزانه‌ی
// start–end است. اگر end زودتر یا مساوی start باشد یعنی بازه از نیمه‌شب
// عبور می‌کند (مثلاً ۲۲:۰۰ تا ۰۲:۰۰). start==end یعنی پوششِ کل شبانه‌روز.
func InDailyWindow(now time.Time, startH, startM, endH, endM int) bool {
	startMin := startH*60 + startM
	endMin := endH*60 + endM
	curMin := now.Hour()*60 + now.Minute()
	if startMin == endMin {
		return true
	}
	if startMin < endMin {
		return curMin >= startMin && curMin < endMin
	}
	return curMin >= startMin || curMin < endMin
}

// CurrentWindowStart لحظه‌ی دقیقِ (با تاریخِ درست) شروعِ بازه‌ی روزانه‌ای
// را برمی‌گرداند که now الان داخل آن است. برای بازه‌های عبورکننده از
// نیمه‌شب، اگر now در بخشِ «بعد از نیمه‌شب» باشد، شروعِ بازه دیروز بوده.
func CurrentWindowStart(now time.Time, startH, startM, endH, endM int) time.Time {
	todayStart := StartTimeOnDay(now, startH, startM)
	startMin := startH*60 + startM
	endMin := endH*60 + endM
	curMin := now.Hour()*60 + now.Minute()
	if startMin >= endMin && curMin < endMin {
		return todayStart.Add(-24 * time.Hour)
	}
	return todayStart
}

// DailyWindowBounds بازه‌ی start–end را برای تاریخِ day می‌سازد (بدون توجه
// به اینکه الان چه ساعتی است) — برای پیش‌بینیِ اسلات‌های یک روزِ دلخواه
// (مثل نمای زمان‌بندی که می‌تواند «فردا»/«دیروز» را هم نشان دهد).
// اگر بازه از نیمه‌شب عبور کند یا کل شبانه‌روز را پوشش دهد، end بیست‌وچهار
// ساعت جلوتر از start برگردانده می‌شود.
func DailyWindowBounds(day time.Time, startH, startM, endH, endM int) (time.Time, time.Time) {
	start := StartTimeOnDay(day, startH, startM)
	end := StartTimeOnDay(day, endH, endM)
	if !end.After(start) {
		end = end.Add(24 * time.Hour)
	}
	return start, end
}

// RotationIndex بر اساس دقیقه‌های سپری‌شده از شروعِ بازه و فاصله‌ی چرخش،
// مشخص می‌کند کدام ایندکس از لیستِ تبلیغ‌ها باید نمایش داده شود.
// rotationMinutes<=0 یعنی بدون چرخش (همیشه ایندکس ۰). numAds<=0 یک حالتِ
// حفاظتی است (چیزی برای چرخاندن نیست).
func RotationIndex(elapsedMinutes, rotationMinutes, numAds int) int {
	if numAds <= 0 {
		return 0
	}
	if rotationMinutes <= 0 {
		return 0
	}
	idx := (elapsedMinutes / rotationMinutes) % numAds
	if idx < 0 {
		idx += numAds // elapsed منفی نباید پیش بیاید، ولی برای اطمینان
	}
	return idx
}

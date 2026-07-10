// handler_schedule.go — نمای زمان‌بندی/تایم‌لاین تبلیغات (بخش ۲.۷ سند
// نیازمندی FEATURES_V2.md).
//
// این نما زمان‌بندیِ واقعیِ ارسال تبلیغ‌ها را روزبه‌روز نشان می‌دهد — فقط
// اسلات‌هایی که واقعاً تبلیغی در آن‌ها قرار است پخش شود، نه کل ۲۴ ساعت.
// چون رزروهای واقعی (Reservation) فقط چند دقیقه پیش از ارسال ساخته
// می‌شوند (برای یادآوری)، این نما به‌جای خواندن از دیتابیس، زمان‌بندیِ
// آینده را از روی تنظیمات خودِ کمپین (بازه‌ی روزانه، فاصله‌ی پست‌ها،
// چرخش) محاسبه می‌کند — تقریبی برای هر کانال، ولی برای دیدِ کلیِ «چه
// زمانی چه تبلیغی» کافی است.
package tgbot

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

// scheduleSlot یک ردیف در نمای زمان‌بندی.
type scheduleSlot struct {
	At           time.Time
	Ad           models.Advertisement
	CampaignName string
}

// projectCampaignSlots اسلات‌های یک کمپین را برای یک روزِ مشخص محاسبه
// می‌کند؛ فقط بر پایه‌ی تنظیمات کمپین، بدون خواندن رزروهای واقعی. محاسبات
// خالصِ بازه/چرخش از internal/models/schedule.go می‌آیند — همان‌هایی که
// scheduler هم برای ارسال واقعی استفاده می‌کند، تا این نما همیشه با
// رفتار واقعی هماهنگ بماند.
func projectCampaignSlots(cm models.Campaign, ads []models.Advertisement, day time.Time) []scheduleSlot {
	if cm.IntervalMinutes < 1 || len(ads) == 0 {
		return nil
	}
	winStart, winEnd := models.DailyWindowBounds(day, cm.StartHour, cm.StartMinute, cm.EndHour, cm.EndMinute)

	maxReplies := 0
	for _, ad := range ads {
		if len(ad.Replies) > maxReplies {
			maxReplies = len(ad.Replies)
		}
	}
	cycleGap := time.Duration(cm.IntervalMinutes*(maxReplies+1)) * time.Minute
	if cycleGap <= 0 {
		return nil
	}

	var slots []scheduleSlot
	for t := winStart; t.Before(winEnd); t = t.Add(cycleGap) {
		elapsed := int(t.Sub(winStart).Minutes())
		idx := models.RotationIndex(elapsed, cm.RotationMinutes, len(ads))
		slots = append(slots, scheduleSlot{At: t, Ad: ads[idx], CampaignName: cm.Name})
		if len(slots) > 300 {
			break // شبکه‌ی ایمنی در برابر تنظیمات نامعقول (مثلاً فاصله‌ی خیلی کوچک)
		}
	}
	return slots
}

// scheduleHome ورود از منوی اصلی (پیام تازه، روزِ «امروز»).
func (h *Handler) scheduleHome(c tele.Context) error {
	return h.renderSchedule(c, false, 0)
}

// scheduleDay ورود از callback پیمایشِ روز (ویرایش پیام موجود).
func (h *Handler) scheduleDay(c tele.Context, arg string) error {
	offset, _ := strconv.Atoi(arg)
	return h.renderSchedule(c, true, offset)
}

func (h *Handler) renderSchedule(c tele.Context, edit bool, dayOffset int) error {
	ctx := context.Background()
	day := time.Now().AddDate(0, 0, dayOffset)

	campaigns, _ := h.store.ListCampaigns(ctx, models.CampaignRunning, 1, listPageSize)
	var slots []scheduleSlot
	for _, cm := range campaigns {
		ads, _ := h.store.ListAdsByCampaign(ctx, cm.ID)
		slots = append(slots, projectCampaignSlots(cm, ads, day)...)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].At.Before(slots[j].At) })

	var b strings.Builder
	fmt.Fprintf(&b, "🗓 <b>زمان‌بندی — %s</b>\n\n", scheduleDayLabel(dayOffset, day))
	if len(campaigns) == 0 {
		b.WriteString("هیچ کمپینِ در حال اجرایی نیست.")
	} else if len(slots) == 0 {
		b.WriteString("در این روز هیچ اسلاتِ زمان‌بندی‌شده‌ای نیست.")
	} else {
		for _, sl := range slots {
			fmt.Fprintf(&b, "⏰ %s — %s<b>%s</b> «%s»\n", sl.At.Format("15:04"), fixedBadge(sl.Ad), sl.Ad.Name, sl.CampaignName)
		}
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			cbBtn(kb, "◀️ دیروز", "sch_day:"+strconv.Itoa(dayOffset-1)),
			cbBtn(kb, "امروز", "sch_day:0"),
			cbBtn(kb, "فردا ▶️", "sch_day:"+strconv.Itoa(dayOffset+1)),
		),
		kb.Row(cbBtn(kb, "🔙 منوی اصلی", "home")),
	)
	if edit {
		return c.Edit(b.String(), tele.ModeHTML, kb)
	}
	return c.Send(b.String(), tele.ModeHTML, kb)
}

func scheduleDayLabel(offset int, day time.Time) string {
	switch offset {
	case 0:
		return "امروز"
	case -1:
		return "دیروز"
	case 1:
		return "فردا"
	default:
		return day.Format("2006-01-02")
	}
}

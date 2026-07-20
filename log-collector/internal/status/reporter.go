package status

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrjvadi/creatorbot/log-collector/internal/store"
	"github.com/mrjvadi/creatorbot/log-collector/internal/telegram"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// updateInterval فاصله‌ی هر بار بازسازی/edit پیامِ داشبورد.
const updateInterval = 30 * time.Second

// downThreshold — اگر بیش از این مدت از یک سرویس heartbeat نرسیده باشد،
// down فرض می‌شود. سه برابرِ فاصله‌ی heartbeat (۲۰ ثانیه در shared/pkg/logger)
// تا یک تأخیرِ شبکه‌ی گذرا باعثِ false-positive نشود.
const downThreshold = 60 * time.Second

// dashboardTopicName نامِ topic ای که این پیام در آن ساخته/edit می‌شود.
const dashboardTopicName = "📊 وضعیت سرویس‌های اصلی"

// mainServices لیستِ سرویس‌های اصلیِ پلتفرم که در داشبورد نمایش داده
// می‌شوند — دقیقاً همان سرویس‌هایی که run.sh راه‌اندازی می‌کند (رجوع
// run.sh ریشه‌ی پروژه)، به‌همراه یک نامِ نمایشیِ خواناتر.
var mainServices = []struct {
	Key   string
	Label string
}{
	{"log-collector", "Log Collector"},
	{"botpay", "BotPay"},
	{"image-registry", "Image Registry"},
	{"license-service", "License Service"},
	{"fraud-engine", "Fraud Engine"},
	{"revenue-service", "Revenue Service"},
	{"community-service", "Community Service"},
	{"webhook-gateway", "Webhook Gateway"},
	{"ads-bot", "Ads Bot"},
	{"agentmanager", "Agent Manager"},
	{"apimanager", "API Manager"},
	{"botmanager", "Bot Manager"},
}

// productBots ربات‌های محصولی که می‌توانند هم‌زمان چند instance (برای چند
// مشتری) داشته باشند — member-bot زیرساخت است ولی همان الگوی deploy
// چندنسخه‌ای را دارد، پس این‌جا هم می‌آید (رجوع
// feedback_memberbot_not_customer_facing در memory: زیرساخت است، نه
// customer-facing، ولی از نظرِ چندنسخه‌ای‌بودن مثل بقیه‌ی این چهارتاست).
// این‌ها به‌جای ✅/⛔ تکی، به‌صورتِ «N از M instance آنلاین» شمرده می‌شوند —
// M همیشه فقط بزرگ می‌شود (تعدادِ instanceهایی که تا الان حداقل یک‌بار دیده
// شده‌اند)، چون تعدادِ واقعیِ instanceِ «باید باشد» را log-collector نمی‌داند.
var productBots = []struct {
	Key   string
	Label string
}{
	{"uploader-bot", "Uploader Bot"},
	{"vpn-bot", "VPN Bot"},
	{"archive-bot", "Archive Bot"},
	{"member-bot", "Member Bot"},
}

// Reporter هر updateInterval یک‌بار وضعیتِ فعلی را رندر و در تلگرام
// edit می‌کند.
type Reporter struct {
	mon   *Monitor
	tg    *telegram.Notifier
	store *store.Store
	log   ports.Logger
}

func NewReporter(mon *Monitor, tg *telegram.Notifier, st *store.Store, log ports.Logger) *Reporter {
	return &Reporter{mon: mon, tg: tg, store: st, log: log}
}

// Run تا پایانِ عمرِ پروسه بلاک می‌شود — باید با go reporter.Run() صدا زده شود.
// اگر تلگرام تنظیم نشده باشد، بی‌صدا برمی‌گردد (مثلِ بقیه‌ی قابلیت‌های
// اختیاریِ تلگرامِ این سرویس).
func (r *Reporter) Run() {
	if r.tg == nil || !r.tg.Enabled() {
		return
	}
	r.tick()
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()
	for range ticker.C {
		r.tick()
	}
}

func (r *Reporter) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.ensureAndSend(ctx, r.render()); err != nil {
		r.log.Error("status dashboard update failed", ports.F("err", err))
	}
}

func (r *Reporter) render() string {
	var b strings.Builder
	b.WriteString("📊 <b>وضعیت سرویس‌های اصلی</b>\n")
	b.WriteString("بروزرسانی: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")

	upCount := 0
	for _, s := range mainServices {
		seen, ok := r.mon.LastSeen(s.Key)
		switch {
		case ok && time.Since(seen) <= downThreshold:
			upCount++
			if started, hasStart := r.mon.StartedAt(s.Key); hasStart && !started.IsZero() {
				b.WriteString(fmt.Sprintf("✅ %s — آپ‌تایم %s\n", s.Label, humanizeDuration(time.Since(started))))
			} else {
				b.WriteString(fmt.Sprintf("✅ %s\n", s.Label))
			}
		case ok:
			b.WriteString(fmt.Sprintf("⛔ %s — آخرین حضور %s پیش\n", s.Label, humanizeDuration(time.Since(seen))))
		default:
			b.WriteString(fmt.Sprintf("❔ %s — هنوز دیده نشده\n", s.Label))
		}
	}
	b.WriteString(fmt.Sprintf("\n%d از %d سرویس بالا\n", upCount, len(mainServices)))

	b.WriteString("\n🤖 <b>ربات‌های محصول (چندنسخه‌ای)</b>\n")
	for _, p := range productBots {
		instances := r.mon.InstanceLastSeen(p.Key)
		if len(instances) == 0 {
			b.WriteString(fmt.Sprintf("❔ %s — هنوز هیچ instance ای دیده نشده\n", p.Label))
			continue
		}
		up := 0
		for _, seen := range instances {
			if time.Since(seen) <= downThreshold {
				up++
			}
		}
		icon := "✅"
		if up < len(instances) {
			icon = "⚠️"
		}
		if up == 0 {
			icon = "⛔"
		}
		b.WriteString(fmt.Sprintf("%s %s — %d از %d instance آنلاین\n", icon, p.Label, up, len(instances)))
	}
	return b.String()
}

// humanizeDuration هم برای «چند وقت پیش down شد» هم برای «چقدر آپ‌تایم
// دارد» استفاده می‌شود — بزرگ‌ترین واحدِ معنادار را نشان می‌دهد (روز/ساعت،
// ساعت/دقیقه، دقیقه/ثانیه، یا فقط ثانیه).
func humanizeDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d ثانیه", int(d.Seconds()))
	case d < time.Hour:
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%d دقیقه", m)
		}
		return fmt.Sprintf("%d دقیقه و %d ثانیه", m, s)
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%d ساعت", h)
		}
		return fmt.Sprintf("%d ساعت و %d دقیقه", h, m)
	default:
		days := int(d.Hours()) / 24
		h := int(d.Hours()) % 24
		if h == 0 {
			return fmt.Sprintf("%d روز", days)
		}
		return fmt.Sprintf("%d روز و %d ساعت", days, h)
	}
}

// ensureAndSend اولین‌بار topic+پیام را می‌سازد و در Mongo ذخیره می‌کند؛
// بارهای بعدی همان پیام را edit می‌کند. اگر edit به هر دلیلی شکست بخورد
// (مثلاً کاربر پیام را پاک کرده)، یک پیامِ جایگزین در همان topic می‌فرستد.
func (r *Reporter) ensureAndSend(ctx context.Context, text string) error {
	dash, ok := r.store.GetStatusDashboard(ctx)
	if !ok {
		threadID, err := r.tg.CreateTopic(ctx, dashboardTopicName)
		if err != nil {
			return err
		}
		msgID, err := r.tg.SendToTopicGetID(ctx, threadID, text)
		if err != nil {
			return err
		}
		return r.store.SaveStatusDashboard(ctx, store.StatusDashboard{MessageThreadID: threadID, MessageID: msgID})
	}

	if err := r.tg.EditMessage(ctx, dash.MessageID, text); err != nil {
		msgID, sendErr := r.tg.SendToTopicGetID(ctx, dash.MessageThreadID, text)
		if sendErr != nil {
			return fmt.Errorf("edit failed (%w) and resend also failed: %v", err, sendErr)
		}
		dash.MessageID = msgID
		return r.store.SaveStatusDashboard(ctx, dash)
	}
	return nil
}

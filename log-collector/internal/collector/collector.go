// Package collector مصرف‌کننده‌ی NATS این سرویس — لاگ‌ها را می‌گیرد، در Mongo
// ذخیره می‌کند، و (اگر تلگرام تنظیم شده باشد) به topic سرویس مربوطه می‌فرستد.
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mrjvadi/creatorbot/log-collector/internal/store"
	"github.com/mrjvadi/creatorbot/log-collector/internal/telegram"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Collector struct {
	store *store.Store
	tg    *telegram.Notifier
	log   ports.Logger

	// minTelegramLevel حداقل سطحی که به تلگرام هم فرستاده می‌شود (warn/error/fatal).
	// همه‌ی سطوح Warn+ همیشه در Mongo ذخیره می‌شوند؛ این فقط فیلتر تلگرام است.
	minTelegramLevel int

	// topicMu از race بین دو goroutine که هم‌زمان اولین لاگ یک سرویس تازه را
	// می‌بینند جلوگیری می‌کند (وگرنه ممکن است دو topic برای یک سرویس ساخته شود).
	topicMu    sync.Mutex
	topicCache map[string]int // service -> message_thread_id (کش حافظه، پشتیبان از Mongo)
}

func New(st *store.Store, tg *telegram.Notifier, log ports.Logger, minTelegramLevel string) *Collector {
	return &Collector{
		store:            st,
		tg:               tg,
		log:              log,
		minTelegramLevel: levelRank(minTelegramLevel),
		topicCache:       map[string]int{},
	}
}

func levelRank(level string) int {
	switch strings.ToLower(level) {
	case "warn":
		return 1
	case "error":
		return 2
	case "fatal":
		return 3
	default:
		return 1 // پیش‌فرض: warn به بالا
	}
}

// Handle یک پیام خام از subject logs.events را پردازش می‌کند.
func (c *Collector) Handle(data []byte) {
	var ev logger.LogEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return
	}
	if ev.Service == "" {
		ev.Service = "unknown"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry := &store.LogEntry{
		Service:   ev.Service,
		Level:     ev.Level,
		Message:   ev.Message,
		Fields:    ev.Fields,
		Timestamp: time.Unix(ev.Timestamp, 0),
	}
	if err := c.store.SaveLog(ctx, entry); err != nil {
		c.log.Error("save log entry failed", ports.F("err", err), ports.F("service", ev.Service))
	}

	if c.tg == nil || !c.tg.Enabled() {
		return
	}
	if levelRank(ev.Level) < c.minTelegramLevel {
		return
	}
	c.notifyTelegram(ctx, ev)
}

func (c *Collector) notifyTelegram(ctx context.Context, ev logger.LogEvent) {
	threadID, err := c.topicFor(ctx, ev.Service)
	if err != nil {
		c.log.Error("get/create telegram topic failed", ports.F("err", err), ports.F("service", ev.Service))
		return
	}

	icon := "⚠️"
	switch strings.ToLower(ev.Level) {
	case "error":
		icon = "🛑"
	case "fatal":
		icon = "💀"
	}

	text := fmt.Sprintf("%s <b>%s</b>\n<code>%s</code>", icon, strings.ToUpper(ev.Level), escapeHTML(ev.Message))
	if len(ev.Fields) > 0 {
		fb, _ := json.MarshalIndent(ev.Fields, "", "  ")
		text += fmt.Sprintf("\n<pre>%s</pre>", escapeHTML(string(fb)))
	}

	if err := c.tg.SendToTopic(ctx, threadID, text); err != nil {
		c.log.Error("send telegram alert failed", ports.F("err", err), ports.F("service", ev.Service))
	}
}

// topicFor شناسه‌ی topic سرویس را برمی‌گرداند — از کش حافظه، بعد Mongo،
// و در نهایت (اگر هیچ‌کدام نبود) با ساخت یک topic تازه در تلگرام.
func (c *Collector) topicFor(ctx context.Context, service string) (int, error) {
	c.topicMu.Lock()
	defer c.topicMu.Unlock()

	if id, ok := c.topicCache[service]; ok {
		return id, nil
	}
	if id, ok := c.store.GetTopicID(ctx, service); ok {
		c.topicCache[service] = id
		return id, nil
	}

	id, err := c.tg.CreateTopic(ctx, service)
	if err != nil {
		return 0, err
	}
	if err := c.store.SaveTopicID(ctx, service, id); err != nil {
		c.log.Error("save topic mapping failed", ports.F("err", err), ports.F("service", service))
	}
	c.topicCache[service] = id
	return id, nil
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

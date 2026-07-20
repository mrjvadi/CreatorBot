// Package status از روی heartbeatِ دوره‌ای هر سرویس (رجوع
// shared/pkg/logger.SubjHeartbeat — خودکار برای هر سرویسی که AttachNATS
// صدا بزند) یک نقشه‌ی «آخرین‌بار کِی دیده شد» نگه می‌دارد، و از رویش یک
// پیامِ داشبوردِ وضعیت در تلگرام می‌سازد که به‌جای پیامِ جدید، هر بار edit
// می‌شود.
package status

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
)

// state آخرین اطلاعاتِ شناخته‌شده از یک سرویس/instance.
type state struct {
	lastSeen  time.Time
	startedAt time.Time
}

// key سرویس + instance (برای سرویس‌های تک‌نسخه‌ای instanceID خالی است).
type key struct {
	service    string
	instanceID string
}

// Monitor نقشه‌ی (سرویس, instance)→وضعیت را نگه می‌دارد؛ Handle مستقیم به
// nc.Subscribe(logger.SubjHeartbeat, ...) داده می‌شود.
type Monitor struct {
	mu    sync.RWMutex
	byKey map[key]state
}

func NewMonitor() *Monitor {
	return &Monitor{byKey: map[key]state{}}
}

// Handle یک پیامِ خام از subject حضور را پردازش می‌کند.
func (m *Monitor) Handle(data []byte) {
	var ev logger.HeartbeatEvent
	if err := json.Unmarshal(data, &ev); err != nil || ev.Service == "" {
		return
	}
	m.mu.Lock()
	m.byKey[key{ev.Service, ev.InstanceID}] = state{
		lastSeen:  time.Unix(ev.Timestamp, 0),
		startedAt: time.Unix(ev.StartedAt, 0),
	}
	m.mu.Unlock()
}

// LastSeen آخرین زمانِ دیده‌شدنِ یک سرویسِ تک‌نسخه‌ای (instanceID خالی) را
// برمی‌گرداند (اگر تا الان هیچ‌وقت هیچ heartbeatی از آن دریافت نشده باشد، ok=false).
func (m *Monitor) LastSeen(service string) (t time.Time, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, found := m.byKey[key{service, ""}]
	return s.lastSeen, found
}

// StartedAt زمانِ شروعِ واقعیِ پروسه‌ی یک سرویسِ تک‌نسخه‌ای را برمی‌گرداند.
func (m *Monitor) StartedAt(service string) (t time.Time, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, found := m.byKey[key{service, ""}]
	return s.startedAt, found
}

// InstanceLastSeen نقشه‌ی instanceID→آخرین‌حضور همه‌ی instanceهایی که تا الان
// از این نوع سرویس دیده شده‌اند برمی‌گرداند — برای سرویس‌های چندنسخه‌ای
// (ربات‌های محصول) که هرکدام instanceID مجزا در heartbeat می‌فرستند.
func (m *Monitor) InstanceLastSeen(service string) map[string]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]time.Time{}
	for k, s := range m.byKey {
		if k.service == service && k.instanceID != "" {
			out[k.instanceID] = s.lastSeen
		}
	}
	return out
}

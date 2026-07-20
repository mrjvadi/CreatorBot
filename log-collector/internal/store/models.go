// Package store ذخیره‌سازی لاگ‌های جمع‌آوری‌شده در MongoDB.
package store

import "time"

// LogEntry یک لاگ Warn/Error/Fatal که از یک سرویس روی logs.events دریافت شده.
type LogEntry struct {
	Service   string         `bson:"service" json:"service"`
	Level     string         `bson:"level" json:"level"` // warn | error | fatal
	Message   string         `bson:"message" json:"message"`
	Fields    map[string]any `bson:"fields,omitempty" json:"fields,omitempty"`
	Timestamp time.Time      `bson:"timestamp" json:"timestamp"`
	// ReceivedAt زمان دریافت توسط log-collector — برای تشخیص تأخیر شبکه/clock skew.
	ReceivedAt time.Time `bson:"received_at" json:"received_at"`
}

// TopicMapping نگاشت هر سرویس به topic ساخته‌شده در سوپرگروه فوروم تلگرام —
// تا برای هر سرویس فقط یک‌بار topic ساخته شود، نه هر پیام.
type TopicMapping struct {
	Service         string `bson:"service" json:"service"`
	MessageThreadID int    `bson:"message_thread_id" json:"message_thread_id"`
}

// StatusDashboard یک سند تک‌نسخه‌ای (singleton، با _id ثابت) که topic و
// message_id پیامِ داشبوردِ وضعیتِ زنده‌ی سرویس‌ها را نگه می‌دارد — تا با
// هر ری‌استارتِ log-collector، همان پیامِ قبلی edit شود، نه یک پیامِ جدید.
type StatusDashboard struct {
	ID              string `bson:"_id" json:"id"`
	MessageThreadID int    `bson:"message_thread_id" json:"message_thread_id"`
	MessageID       int    `bson:"message_id" json:"message_id"`
}

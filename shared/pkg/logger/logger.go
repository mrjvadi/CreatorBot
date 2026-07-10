// Package logger implements ports.Logger using go.uber.org/zap.
// To swap to slog: implement ports.Logger in a new package and wire in main.go.
package logger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// SubjLogEvents — همه‌ی سرویس‌ها لاگ‌های سطح Warn به بالا را روی همین
// subject منتشر می‌کنند؛ سرویس log-collector این‌ها را جمع می‌کند.
// Debug/Info هرگز روی NATS منتشر نمی‌شوند — فیلتر در همان لحظه‌ی تولید لاگ
// انجام می‌شود تا حجم پیام روی NATS زیاد نشود.
const SubjLogEvents = "logs.events"

// LogEvent ساختار پیامی که برای هر لاگ Warn/Error/Fatal روی NATS می‌رود.
type LogEvent struct {
	Service   string         `json:"service"`
	Level     string         `json:"level"` // warn | error | fatal
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	Timestamp int64          `json:"ts"`
}

// ZapLogger wraps *zap.Logger and implements ports.Logger.
type ZapLogger struct {
	z *zap.Logger

	// natsSink اختیاری — اگر با AttachNATS تنظیم شود، هر Warn/Error/Fatal
	// علاوه بر خروجی محلی، روی NATS هم منتشر می‌شود (best-effort، هرگز
	// نباید چیزی را block یا panic کند — لاگ‌گیری نباید خودش خطرِ سرویس شود).
	nc          *natsclient.Client
	serviceName string
}

var _ ports.Logger = (*ZapLogger)(nil)

// New creates a ZapLogger.
// production=true: JSON output, no caller info in debug.
// production=false: console output, colored, with caller.
func New(production bool) (*ZapLogger, error) {
	var z *zap.Logger
	var err error
	if production {
		z, err = zap.NewProduction()
	} else {
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		z, err = cfg.Build()
	}
	if err != nil {
		return nil, err
	}
	return &ZapLogger{z: z}, nil
}

// MustNew panics if logger creation fails.
func MustNew(production bool) *ZapLogger {
	l, err := New(production)
	if err != nil {
		panic(err)
	}
	return l
}

func (l *ZapLogger) Debug(msg string, fields ...ports.Field) { l.z.Debug(msg, toZap(fields)...) }
func (l *ZapLogger) Info(msg string, fields ...ports.Field)  { l.z.Info(msg, toZap(fields)...) }

func (l *ZapLogger) Warn(msg string, fields ...ports.Field) {
	l.z.Warn(msg, toZap(fields)...)
	l.publish("warn", msg, fields)
}

func (l *ZapLogger) Error(msg string, fields ...ports.Field) {
	l.z.Error(msg, toZap(fields)...)
	l.publish("error", msg, fields)
}

func (l *ZapLogger) Fatal(msg string, fields ...ports.Field) {
	// نکته: l.z.Fatal خودش os.Exit(1) صدا می‌زند، پس باید publish را قبل از
	// آن انجام دهیم وگرنه هرگز اجرا نمی‌شود.
	l.publish("fatal", msg, fields)
	l.z.Fatal(msg, toZap(fields)...)
}

func (l *ZapLogger) With(fields ...ports.Field) ports.Logger {
	return &ZapLogger{z: l.z.With(toZap(fields)...), nc: l.nc, serviceName: l.serviceName}
}

// AttachNATS یک sink اختیاری به NATS وصل می‌کند — از این لحظه به بعد، هر
// Warn/Error/Fatal علاوه بر خروجی محلی (stdout/فایل)، روی SubjLogEvents هم
// منتشر می‌شود تا سرویس log-collector آن را جمع کند. صدا زدن این متد کاملاً
// اختیاری است؛ بدون آن، رفتار لاگر دقیقاً مثل قبل (فقط محلی) باقی می‌ماند.
func (l *ZapLogger) AttachNATS(nc *natsclient.Client, serviceName string) {
	l.nc = nc
	l.serviceName = serviceName
}

// publish تلاش می‌کند لاگ را روی NATS منتشر کند — کاملاً best-effort:
// nil-safe (اگر AttachNATS صدا زده نشده باشد کاری نمی‌کند)، panic-safe
// (recover می‌کند تا یک مشکل غیرمنتظره در marshal/publish هرگز خودِ سرویس
// را crash نکند)، و هرگز مسیر اصلی برنامه را block نمی‌کند چون NATS core
// publish خودش async و بدون انتظار ack است.
func (l *ZapLogger) publish(level, msg string, fields []ports.Field) {
	if l.nc == nil {
		return
	}
	defer func() { _ = recover() }()

	fm := make(map[string]any, len(fields))
	for _, f := range fields {
		// خیلی از فراخوانی‌ها ports.F("err", err) هستند؛ نوع error معمولاً
		// struct بدون فیلد export‌شده است و json.Marshal آن را {} خالی
		// می‌کند — این‌جا صریحاً به رشته‌ی Error() تبدیل می‌شود تا در Mongo/
		// تلگرام قابل‌خواندن باشد.
		if err, ok := f.Value.(error); ok {
			fm[f.Key] = err.Error()
			continue
		}
		fm[f.Key] = f.Value
	}
	_ = l.nc.PublishCore(SubjLogEvents, LogEvent{
		Service:   l.serviceName,
		Level:     level,
		Message:   msg,
		Fields:    fm,
		Timestamp: time.Now().Unix(),
	})
}

func toZap(fields []ports.Field) []zap.Field {
	out := make([]zap.Field, len(fields))
	for i, f := range fields {
		out[i] = zap.Any(f.Key, f.Value)
	}
	return out
}

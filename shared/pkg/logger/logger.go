// Package logger implements ports.Logger using go.uber.org/zap.
// To swap to slog: implement ports.Logger in a new package and wire in main.go.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ZapLogger wraps *zap.Logger and implements ports.Logger.
type ZapLogger struct {
	z *zap.Logger
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
func (l *ZapLogger) Warn(msg string, fields ...ports.Field)  { l.z.Warn(msg, toZap(fields)...) }
func (l *ZapLogger) Error(msg string, fields ...ports.Field) { l.z.Error(msg, toZap(fields)...) }
func (l *ZapLogger) Fatal(msg string, fields ...ports.Field) { l.z.Fatal(msg, toZap(fields)...) }

func (l *ZapLogger) With(fields ...ports.Field) ports.Logger {
	return &ZapLogger{z: l.z.With(toZap(fields)...)}
}

func toZap(fields []ports.Field) []zap.Field {
	out := make([]zap.Field, len(fields))
	for i, f := range fields {
		out[i] = zap.Any(f.Key, f.Value)
	}
	return out
}

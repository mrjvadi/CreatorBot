package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// noopLogger یک ports.Logger خاموش برای تست است — هیچ‌جا چاپ نمی‌کند.
type noopLogger struct{}

func (noopLogger) Debug(string, ...ports.Field)       {}
func (noopLogger) Info(string, ...ports.Field)        {}
func (noopLogger) Warn(string, ...ports.Field)        {}
func (noopLogger) Error(string, ...ports.Field)       {}
func (noopLogger) Fatal(string, ...ports.Field)       {}
func (l noopLogger) With(...ports.Field) ports.Logger { return l }

var _ ports.Logger = noopLogger{}

// fakeImageChecker پیاده‌سازی آزمایشی ImageChecker — بدون HTTP واقعی.
type fakeImageChecker struct {
	allowed bool
	err     error
	// calls هر (name, tag) که واقعاً پرسیده شده را ثبت می‌کند.
	calls []string
}

func (f *fakeImageChecker) IsAllowed(_ context.Context, name, tag string) (bool, error) {
	f.calls = append(f.calls, name+":"+tag)
	return f.allowed, f.err
}

func TestSplitImageRef(t *testing.T) {
	tests := []struct {
		ref      string
		wantName string
		wantTag  string
		wantOK   bool
	}{
		{"nginx:latest", "nginx", "latest", true},
		{"myregistry.io:5000/myapp:v1", "myregistry.io:5000/myapp", "v1", true},
		{"nginx", "", "", false},        // بدون تگ — رد می‌شود
		{"nginx:", "", "", false},       // تگ خالی — رد می‌شود
		{"", "", "", false},             // ref خالی
		{":latest", "", "latest", true}, // نام خالی ولی تگ دارد — splitImageRef فقط پارس می‌کند
	}

	for _, tt := range tests {
		name, tag, ok := splitImageRef(tt.ref)
		if ok != tt.wantOK {
			t.Errorf("splitImageRef(%q) ok = %v, want %v", tt.ref, ok, tt.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if name != tt.wantName || tag != tt.wantTag {
			t.Errorf("splitImageRef(%q) = (%q, %q), want (%q, %q)", tt.ref, name, tag, tt.wantName, tt.wantTag)
		}
	}
}

func TestIsImageAllowed_NoTag_RejectsWithoutCallingRegistry(t *testing.T) {
	checker := &fakeImageChecker{allowed: true}
	c := &Client{log: noopLogger{}, registry: checker}

	if got := c.isImageAllowed(context.Background(), "nginx"); got {
		t.Error("expected false for ref without tag")
	}
	if len(checker.calls) != 0 {
		t.Errorf("expected registry not to be called for an invalid ref, got calls=%v", checker.calls)
	}
}

func TestIsImageAllowed_Allowed(t *testing.T) {
	checker := &fakeImageChecker{allowed: true}
	c := &Client{log: noopLogger{}, registry: checker}

	if got := c.isImageAllowed(context.Background(), "myapp:v1"); !got {
		t.Error("expected true when registry allows the image")
	}
	if len(checker.calls) != 1 || checker.calls[0] != "myapp:v1" {
		t.Errorf("expected registry to be called once with myapp:v1, got %v", checker.calls)
	}
}

func TestIsImageAllowed_Denied(t *testing.T) {
	checker := &fakeImageChecker{allowed: false}
	c := &Client{log: noopLogger{}, registry: checker}

	if got := c.isImageAllowed(context.Background(), "myapp:v1"); got {
		t.Error("expected false when registry denies the image")
	}
}

// TestIsImageAllowed_FailClosedOnError پوشش مهم‌ترین قرارداد امنیتی این فایل:
// هر خطا از image-registry (شبکه قطع، IP رد شده، ...) باید به رد deploy
// منجر شود، نه اجازه‌ی پیش‌فرض.
func TestIsImageAllowed_FailClosedOnError(t *testing.T) {
	checker := &fakeImageChecker{allowed: true, err: errors.New("image-registry unreachable")}
	c := &Client{log: noopLogger{}, registry: checker}

	if got := c.isImageAllowed(context.Background(), "myapp:v1"); got {
		t.Error("expected fail-closed (false) when the registry client returns an error, even if allowed=true")
	}
}

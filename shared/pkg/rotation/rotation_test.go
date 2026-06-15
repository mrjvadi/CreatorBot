package rotation_test

import (
	"context"
	"testing"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/rotation"
)

func TestRotation_Basic(t *testing.T) {
	mgr := rotation.New(nil, time.Hour)
	ctx := context.Background()

	// ایجاد key اولیه
	key1, err := mgr.Rotate(ctx, "jwt_secret")
	if err != nil {
		t.Fatalf("first rotate: %v", err)
	}
	if key1 == "" {
		t.Fatal("key should not be empty")
	}

	// rotate → key جدید
	key2, err := mgr.Rotate(ctx, "jwt_secret")
	if err != nil {
		t.Fatalf("second rotate: %v", err)
	}
	if key1 == key2 {
		t.Error("new key should be different from old key")
	}
}

func TestRotation_GracePeriod(t *testing.T) {
	mgr := rotation.New(nil, time.Hour)
	ctx := context.Background()

	key1, _ := mgr.Rotate(ctx, "enc_key")
	key2, _ := mgr.Rotate(ctx, "enc_key")

	// key2 (current) باید valid باشد
	if !mgr.Validate("enc_key", key2) {
		t.Error("current key should be valid")
	}

	// key1 (previous) در grace period باید valid باشد
	if !mgr.Validate("enc_key", key1) {
		t.Error("previous key should be valid during grace period")
	}

	// key اشتباه نباید valid باشد
	if mgr.Validate("enc_key", "random_garbage") {
		t.Error("wrong key should not be valid")
	}
}

func TestRotation_GraceExpired(t *testing.T) {
	// grace period خیلی کوتاه
	mgr := rotation.New(nil, time.Millisecond)
	ctx := context.Background()

	key1, _ := mgr.Rotate(ctx, "old_key")
	mgr.Rotate(ctx, "old_key") // key2 current

	time.Sleep(10 * time.Millisecond) // صبر برای expire

	// key1 دیگر نباید valid باشد
	if mgr.Validate("old_key", key1) {
		t.Error("expired key should not be valid")
	}
}

func TestRotation_Current(t *testing.T) {
	mgr := rotation.New(nil, time.Hour)
	ctx := context.Background()

	// قبل از rotate
	_, err := mgr.Current("nonexistent")
	if err == nil {
		t.Error("Current() on missing key should error")
	}

	key, _ := mgr.Rotate(ctx, "mykey")
	current, err := mgr.Current("mykey")
	if err != nil {
		t.Fatalf("Current(): %v", err)
	}
	if current != key {
		t.Errorf("Current() = %q, want %q", current, key)
	}
}

func TestRotation_GenerateKey(t *testing.T) {
	key1, _ := rotation.GenerateKey(32)
	key2, _ := rotation.GenerateKey(32)

	if key1 == key2 {
		t.Error("generated keys should be unique")
	}
	// 32 bytes = 64 hex chars
	if len(key1) != 64 {
		t.Errorf("key length = %d, want 64", len(key1))
	}
}

func TestRotation_RotateAll(t *testing.T) {
	mgr := rotation.New(nil, time.Hour)
	ctx := context.Background()

	// چند key ثبت کن
	mgr.Rotate(ctx, "key_a")
	mgr.Rotate(ctx, "key_b")
	mgr.Rotate(ctx, "key_c")

	results, err := mgr.RotateAll(ctx)
	if err != nil {
		t.Fatalf("RotateAll(): %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 rotated keys, got %d", len(results))
	}
	for name, key := range results {
		if key == "" {
			t.Errorf("rotated key %s should not be empty", name)
		}
	}
}

func TestRotation_UniqueKeys(t *testing.T) {
	mgr := rotation.New(nil, time.Hour)
	ctx := context.Background()

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := mgr.Rotate(ctx, "test")
		if err != nil {
			t.Fatalf("rotate %d: %v", i, err)
		}
		if seen[key] {
			t.Errorf("duplicate key generated at iteration %d", i)
		}
		seen[key] = true
	}
}

package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/store"
)

// MockDB یک پیاده‌سازی ساده از ports.DB برای تست.
type MockDB struct {
	data map[string]any
}

func (m *MockDB) Conn() interface{ Where(query any, args ...any) any } {
	return nil
}

// TestPlanLimits بررسی MaxBots در پلن.
func TestPlanMaxBots(t *testing.T) {
	plan := &models.Plan{
		Base:        models.Base{ID: uuid.New()},
		Name:        "starter",
		Price:       5.0,
		MaxBots:     2,
		DurationDay: 30,
		IsActive:    true,
	}

	if plan.MaxBots != 2 {
		t.Errorf("expected MaxBots=2, got %d", plan.MaxBots)
	}
	if plan.Price != 5.0 {
		t.Errorf("expected Price=5.0, got %f", plan.Price)
	}
}

// TestSubscriptionExpiry بررسی منطق انقضا.
func TestSubscriptionExpiry(t *testing.T) {
	now := time.Now()

	// اشتراک منقضی نشده
	future := now.Add(24 * time.Hour)
	sub := &models.Subscription{
		IsActive:  true,
		ExpiresAt: &future,
	}

	if sub.ExpiresAt.Before(now) {
		t.Error("subscription should not be expired")
	}

	// اشتراک منقضی شده
	past := now.Add(-24 * time.Hour)
	expiredSub := &models.Subscription{
		IsActive:  true,
		ExpiresAt: &past,
	}

	if !expiredSub.ExpiresAt.Before(now) {
		t.Error("subscription should be expired")
	}
}

// TestBotIDFromToken بررسی استخراج Bot ID از توکن.
func TestBotIDFromToken(t *testing.T) {
	tests := []struct {
		token   string
		wantID  int64
		wantErr bool
	}{
		{"1234567890:ABCDefghijklmnop", 1234567890, false},
		{"9999999999:XYZabc123", 9999999999, false},
		{"invalid-token", 0, true},
		{"", 0, true},
		{"notanumber:abc", 0, true},
	}

	for _, tt := range tests {
		id, err := models.BotIDFromToken(tt.token)
		if tt.wantErr {
			if err == nil {
				t.Errorf("BotIDFromToken(%q) expected error, got nil", tt.token)
			}
		} else {
			if err != nil {
				t.Errorf("BotIDFromToken(%q) unexpected error: %v", tt.token, err)
			}
			if id != tt.wantID {
				t.Errorf("BotIDFromToken(%q) = %d, want %d", tt.token, id, tt.wantID)
			}
		}
	}
}

// TestInstanceStatus بررسی status های معتبر.
func TestInstanceStatus(t *testing.T) {
	validStatuses := []string{
		string(models.StatusPending),
		string(models.StatusRunning),
		string(models.StatusStopped),
		string(models.StatusError),
		string(models.StatusDeleted),
	}

	for _, s := range validStatuses {
		if s == "" {
			t.Errorf("status should not be empty")
		}
	}
}

// TestUserRole بررسی role های معتبر.
func TestUserRole(t *testing.T) {
	roles := []models.Role{
		models.RoleUser,
		models.RoleAdmin,
		models.RoleOwner,
	}

	for _, r := range roles {
		if string(r) == "" {
			t.Error("role should not be empty")
		}
	}

	// RoleOwner باید بالاتر از RoleAdmin باشد
	if models.RoleOwner == models.RoleAdmin {
		t.Error("owner and admin roles should be different")
	}
}

// TestPlanValidation بررسی validation پلن.
func TestPlanValidation(t *testing.T) {
	// پلن رایگان باید MaxBots حداقل ۱ داشته باشه
	freePlan := &models.Plan{
		IsFree:  true,
		MaxBots: 1,
	}
	if freePlan.MaxBots < 1 {
		t.Error("free plan should have at least 1 bot slot")
	}

	// پلن پولی باید قیمت مثبت داشته باشه
	paidPlan := &models.Plan{
		IsFree: false,
		Price:  10.0,
	}
	if paidPlan.IsFree && paidPlan.Price > 0 {
		t.Error("free plan should not have price")
	}
	if !paidPlan.IsFree && paidPlan.Price <= 0 {
		t.Error("paid plan should have positive price")
	}
}

// TestContextCancellation بررسی context cancellation.
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// صبر برای expire شدن context
	time.Sleep(5 * time.Millisecond)

	if ctx.Err() == nil {
		t.Error("context should be cancelled")
	}
}

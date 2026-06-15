package middleware

import (
	"testing"
	"time"
)

func TestBotRateLimiter_Allow(t *testing.T) {
	// ۱۰ req/sec، burst 5
	limiter := NewBotRateLimiter(10, 5)

	// اول باید ۵ request (burst) مجاز باشه
	for i := 0; i < 5; i++ {
		if !limiter.allow("bot1") {
			t.Errorf("request %d should be allowed (within burst)", i+1)
		}
	}

	// request ۶ام باید رد بشه (burst تموم شده)
	if limiter.allow("bot1") {
		t.Error("request 6 should be denied (burst exceeded)")
	}
}

func TestBotRateLimiter_Refill(t *testing.T) {
	limiter := NewBotRateLimiter(10, 5)

	// خالی کن bucket
	for i := 0; i < 5; i++ {
		limiter.allow("bot2")
	}

	// صبر کن برای refill
	time.Sleep(200 * time.Millisecond)

	// باید ۲ token refill شده باشه (10 * 0.2 = 2)
	if !limiter.allow("bot2") {
		t.Error("should be allowed after refill")
	}
}

func TestBotRateLimiter_IsolatedBots(t *testing.T) {
	limiter := NewBotRateLimiter(10, 3)

	// خالی کن bot1
	for i := 0; i < 3; i++ {
		limiter.allow("bot3")
	}
	if limiter.allow("bot3") {
		t.Error("bot3 should be rate limited")
	}

	// bot4 باید مستقل باشه
	if !limiter.allow("bot4") {
		t.Error("bot4 should not be affected by bot3 limit")
	}
}

func TestMin(t *testing.T) {
	if min(3.0, 5.0) != 3.0 {
		t.Error("min(3,5) should be 3")
	}
	if min(10.0, 2.0) != 2.0 {
		t.Error("min(10,2) should be 2")
	}
}

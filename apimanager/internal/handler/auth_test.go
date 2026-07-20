package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"testing"
	"time"
)

func telegramTestHash(fields map[string]string, token string) string {
	keys := make([]string, 0, len(fields))
	for key, value := range fields {
		if value != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+fields[key])
	}
	secret := sha256.Sum256([]byte(token))
	mac := hmac.New(sha256.New, secret[:])
	_, _ = mac.Write([]byte(strings.Join(pairs, "\n")))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyTelegramAuthOfficialAndLegacyID(t *testing.T) {
	const token = "123456:test-token"
	for _, idKey := range []string{"id", "telegram_id"} {
		fields := map[string]string{
			idKey:        "7631742375",
			"first_name": "javad",
			"auth_date":  "1784099441",
		}
		hash := telegramTestHash(fields, token)
		if !verifyTelegramAuth(fields, hash, token) {
			t.Fatalf("valid %s signature was rejected", idKey)
		}
		if verifyTelegramAuth(fields, hash, "different-token") {
			t.Fatalf("%s signature accepted with a different token", idKey)
		}
	}
}

func TestValidTelegramAuthTime(t *testing.T) {
	now := time.Unix(1_784_099_441, 0)
	tests := []struct {
		name string
		date int64
		want bool
	}{
		{name: "fresh", date: now.Add(-time.Minute).Unix(), want: true},
		{name: "max future skew", date: now.Add(5 * time.Minute).Unix(), want: true},
		{name: "too old", date: now.Add(-maxAuthAge - time.Second).Unix(), want: false},
		{name: "future", date: now.Add(5*time.Minute + time.Second).Unix(), want: false},
		{name: "missing", date: 0, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validTelegramAuthTime(now, test.date); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

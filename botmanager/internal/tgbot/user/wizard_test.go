package user

import (
	"testing"
)

// TestExtractBotID بررسی استخراج Bot ID از توکن.
func TestExtractBotID(t *testing.T) {
	tests := []struct {
		token   string
		wantID  int64
		wantErr bool
	}{
		{"1234567890:ABCDefghijklmno", 1234567890, false},
		{"9876543210:XYZ_abc-123", 9876543210, false},
		{"", 0, true},
		{"invalid", 0, true},
		{"abc:def", 0, true},
		{":abc", 0, true},
	}

	for _, tt := range tests {
		id, err := extractBotID(tt.token)
		if tt.wantErr {
			if err == nil {
				t.Errorf("extractBotID(%q): expected error, got nil", tt.token)
			}
		} else {
			if err != nil {
				t.Errorf("extractBotID(%q): unexpected error: %v", tt.token, err)
			}
			if id != tt.wantID {
				t.Errorf("extractBotID(%q) = %d, want %d", tt.token, id, tt.wantID)
			}
		}
	}
}

// TestTokenValidation بررسی اعتبارسنجی توکن تلگرام.
func TestTokenValidation(t *testing.T) {
	validTokens := []string{
		"1234567890:ABCDefghijklmnopqrstuvwxyz",
		"9999999999:AAAAAAAAAAAAAAAAAAAAAAAAAAA",
	}
	for _, token := range validTokens {
		_, err := extractBotID(token)
		if err != nil {
			t.Errorf("valid token %q should not fail: %v", token, err)
		}
	}

	invalidTokens := []string{
		"",
		"notavalidtoken",
		"12345",
		"abc:def",
	}
	for _, token := range invalidTokens {
		_, err := extractBotID(token)
		if err == nil {
			t.Errorf("invalid token %q should fail", token)
		}
	}
}

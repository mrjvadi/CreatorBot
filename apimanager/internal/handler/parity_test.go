package handler

import (
	"encoding/hex"
	"testing"
)

func TestSecureRandomHex(t *testing.T) {
	first, err := secureRandomHex(32)
	if err != nil {
		t.Fatalf("secureRandomHex: %v", err)
	}
	second, err := secureRandomHex(32)
	if err != nil {
		t.Fatalf("secureRandomHex second call: %v", err)
	}
	if len(first) != 64 {
		t.Fatalf("length = %d, want 64", len(first))
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Fatalf("result is not hex: %v", err)
	}
	if first == second {
		t.Fatal("two generated credentials must not be equal")
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{name: "first", values: []string{"container-id", "fallback"}, want: "container-id"},
		{name: "fallback", values: []string{"", "container-name"}, want: "container-name"},
		{name: "ignore whitespace", values: []string{"  ", "name"}, want: "name"},
		{name: "empty", values: nil, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstNonEmpty(tt.values...); got != tt.want {
				t.Fatalf("firstNonEmpty() = %q, want %q", got, tt.want)
			}
		})
	}
}

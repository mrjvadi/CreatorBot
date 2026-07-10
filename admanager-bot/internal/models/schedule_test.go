package models

import (
	"testing"
	"time"
)

func at(hour, minute int) time.Time {
	return time.Date(2026, 7, 4, hour, minute, 0, 0, time.UTC)
}

func TestInDailyWindow(t *testing.T) {
	cases := []struct {
		name           string
		now            time.Time
		sh, sm, eh, em int
		want           bool
	}{
		{"normal window inside", at(12, 0), 8, 0, 23, 0, true},
		{"normal window before start", at(7, 0), 8, 0, 23, 0, false},
		{"normal window after end", at(23, 30), 8, 0, 23, 0, false},
		{"normal window exactly at start", at(8, 0), 8, 0, 23, 0, true},
		{"normal window exactly at end (exclusive)", at(23, 0), 8, 0, 23, 0, false},
		{"wraparound inside before midnight", at(23, 30), 22, 0, 2, 0, true},
		{"wraparound inside after midnight", at(1, 0), 22, 0, 2, 0, true},
		{"wraparound outside", at(12, 0), 22, 0, 2, 0, false},
		{"whole day coverage (start==end)", at(3, 17), 0, 0, 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := InDailyWindow(c.now, c.sh, c.sm, c.eh, c.em)
			if got != c.want {
				t.Errorf("InDailyWindow(%02d:%02d, %02d:%02d-%02d:%02d) = %v, want %v",
					c.now.Hour(), c.now.Minute(), c.sh, c.sm, c.eh, c.em, got, c.want)
			}
		})
	}
}

func TestCurrentWindowStart(t *testing.T) {
	cases := []struct {
		name           string
		now            time.Time
		sh, sm, eh, em int
		wantDayOffset  int // نسبت به روزِ now
		wantHour       int
		wantMinute     int
	}{
		{"normal window same day", at(12, 0), 8, 0, 23, 0, 0, 8, 0},
		{"wraparound before midnight", at(23, 30), 22, 0, 2, 0, 0, 22, 0},
		{"wraparound after midnight belongs to yesterday", at(1, 0), 22, 0, 2, 0, -1, 22, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CurrentWindowStart(c.now, c.sh, c.sm, c.eh, c.em)
			wantDay := c.now.AddDate(0, 0, c.wantDayOffset)
			if got.Day() != wantDay.Day() || got.Hour() != c.wantHour || got.Minute() != c.wantMinute {
				t.Errorf("CurrentWindowStart(%v) = %v, want day=%d %02d:%02d",
					c.now, got, wantDay.Day(), c.wantHour, c.wantMinute)
			}
		})
	}
}

func TestDailyWindowBounds(t *testing.T) {
	day := at(0, 0)

	start, end := DailyWindowBounds(day, 8, 0, 23, 0)
	if start.Hour() != 8 || end.Hour() != 23 || end.Sub(start) != 15*time.Hour {
		t.Errorf("normal window bounds wrong: start=%v end=%v", start, end)
	}

	start, end = DailyWindowBounds(day, 22, 0, 2, 0)
	if start.Hour() != 22 || !end.After(start) || end.Sub(start) != 4*time.Hour {
		t.Errorf("wraparound window bounds wrong: start=%v end=%v (expected 4h span)", start, end)
	}

	start, end = DailyWindowBounds(day, 0, 0, 0, 0)
	if end.Sub(start) != 24*time.Hour {
		t.Errorf("whole-day window should span 24h, got %v", end.Sub(start))
	}
}

func TestRotationIndex(t *testing.T) {
	cases := []struct {
		name                      string
		elapsed, rotation, numAds int
		want                      int
	}{
		{"no rotation always first", 999, 0, 5, 0},
		{"zero ads is safe", 10, 5, 0, 0},
		{"first slot", 0, 30, 3, 0},
		{"second slot", 30, 30, 3, 1},
		{"wraps around", 90, 30, 3, 0},
		{"partway through a slot", 45, 30, 3, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := RotationIndex(c.elapsed, c.rotation, c.numAds)
			if got != c.want {
				t.Errorf("RotationIndex(%d, %d, %d) = %d, want %d", c.elapsed, c.rotation, c.numAds, got, c.want)
			}
		})
	}
}

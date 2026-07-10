package models

import "testing"

func TestNormalizeDigits(t *testing.T) {
	cases := map[string]string{
		"۲۳:۰۸":          "23:08",
		"٢٣:٠٨":          "23:08",
		"23:08":          "23:08",
		"۱۰\n۵":          "10\n5",
		"no-digits-here": "no-digits-here",
	}
	for in, want := range cases {
		if got := NormalizeDigits(in); got != want {
			t.Errorf("NormalizeDigits(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseClock(t *testing.T) {
	cases := []struct {
		in   string
		h, m int
		ok   bool
	}{
		{"23:08", 23, 8, true},
		{"9", 9, 0, true},
		{"00:00", 0, 0, true},
		{"24:00", 0, 0, false},  // ساعت نامعتبر
		{"12:60", 0, 0, false},  // دقیقه نامعتبر
		{"-1:00", 0, 0, false},  // منفی
		{"abc", 0, 0, false},    // غیرعددی
		{" 8 : 5 ", 8, 5, true}, // فاصله‌ی اضافه باید نادیده گرفته شود
	}
	for _, c := range cases {
		h, m, ok := ParseClock(c.in)
		if ok != c.ok || (ok && (h != c.h || m != c.m)) {
			t.Errorf("ParseClock(%q) = (%d, %d, %v), want (%d, %d, %v)", c.in, h, m, ok, c.h, c.m, c.ok)
		}
	}
}

func TestParseClockRange(t *testing.T) {
	sh, sm, eh, em, ok := ParseClockRange("23:08-03:00")
	if !ok || sh != 23 || sm != 8 || eh != 3 || em != 0 {
		t.Errorf("ParseClockRange(23:08-03:00) = (%d,%d,%d,%d,%v)", sh, sm, eh, em, ok)
	}

	sh, sm, eh, em, ok = ParseClockRange("08:00")
	if !ok || sh != 8 || sm != 0 || eh != 8 || em != 0 {
		t.Errorf("ParseClockRange(08:00) without a dash should mean whole-day (start==end), got (%d,%d,%d,%d,%v)", sh, sm, eh, em, ok)
	}

	if _, _, _, _, ok := ParseClockRange("bad-input"); ok {
		t.Error("ParseClockRange(bad-input) should fail")
	}
}

func TestParseSchedule(t *testing.T) {
	sh, sm, eh, em, interval, del, rot, ok := ParseSchedule("23:08-03:00\n10\n60\n120")
	if !ok || sh != 23 || sm != 8 || eh != 3 || em != 0 || interval != 10 || del != 60 || rot != 120 {
		t.Fatalf("unexpected parse result: %d %d %d %d %d %d %d %v", sh, sm, eh, em, interval, del, rot, ok)
	}

	// ارقام فارسی هم باید کار کند.
	_, _, _, _, interval, _, _, ok = ParseSchedule("۲۳:۰۸-۰۳:۰۰\n۱۰\n۶۰\n۱۲۰")
	if !ok || interval != 10 {
		t.Errorf("Persian digits should parse the same, got interval=%d ok=%v", interval, ok)
	}

	// تعداد خط اشتباه
	if _, _, _, _, _, _, _, ok := ParseSchedule("23:08-03:00\n10\n60"); ok {
		t.Error("ParseSchedule with only 3 lines should fail")
	}

	// فاصله‌ی صفر یا منفی نامعتبر است
	if _, _, _, _, _, _, _, ok := ParseSchedule("08:00-20:00\n0\n60\n0"); ok {
		t.Error("ParseSchedule with interval=0 should fail (interval must be >= 1)")
	}

	// عمر چرخه یا چرخشِ منفی نامعتبر است
	if _, _, _, _, _, _, _, ok := ParseSchedule("08:00-20:00\n10\n-1\n0"); ok {
		t.Error("ParseSchedule with negative deleteAfter should fail")
	}
}

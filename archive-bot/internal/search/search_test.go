package search

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"كتاب يادگيري":       "کتاب یادگیری",
		"برنامه‌نویسی":        "برنامهنویسی", // ZWNJ حذف می‌شود (نه جایگزینی با space)
		"سَلامٌ عَلَیکُم":       "سلام علیکم",
		"  چند   فاصله  ":    "چند فاصله",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTrigrams(t *testing.T) {
	tg := Trigrams("abc")
	want := []string{" ab", "abc", "bc "}
	if len(tg) != len(want) {
		t.Fatalf("Trigrams(\"abc\") = %v, want %v", tg, want)
	}
	for i, w := range want {
		if tg[i] != w {
			t.Errorf("Trigrams(\"abc\")[%d] = %q, want %q", i, tg[i], w)
		}
	}

	if got := Trigrams(""); got != nil {
		t.Errorf("Trigrams(\"\") = %v, want nil", got)
	}
	// رشته‌ی کوتاه‌تر از یک trigram (با padding هم به ۳ نمی‌رسد؟ " a " دقیقاً ۳ کاراکتر است)
	if got := Trigrams("a"); len(got) != 1 {
		t.Errorf("Trigrams(\"a\") = %v, want exactly 1 trigram", got)
	}
}

func TestSimilarity(t *testing.T) {
	a := Trigrams(Normalize("آموزش زبان گو"))
	b := Trigrams(Normalize("آموزش زبان گو"))
	if sc := Similarity(a, b); sc != 1.0 {
		t.Errorf("identical strings: Similarity = %v, want 1.0", sc)
	}

	c := Trigrams(Normalize("فیلم سینمایی اکشن"))
	if sc := Similarity(a, c); sc > 0.1 {
		t.Errorf("unrelated strings: Similarity = %v, expected <= 0.1", sc)
	}

	if sc := Similarity(nil, b); sc != 0 {
		t.Errorf("empty set: Similarity = %v, want 0", sc)
	}

	// شباهتِ نسبی: دو رشته‌ی نزدیک باید امتیازی بینِ صفر و یک بگیرند.
	d := Trigrams(Normalize("آموزش زبان برنامه‌نویسی گو"))
	if sc := Similarity(a, d); sc <= 0 || sc >= 1 {
		t.Errorf("partial overlap: Similarity = %v, expected strictly between 0 and 1", sc)
	}
}

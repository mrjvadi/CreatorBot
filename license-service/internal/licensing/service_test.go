package licensing

import "testing"

// TestIsTestToken بررسی می‌کند بایپسِ لایسنسِ تستی فقط دقیقاً وقتی فعال است که
// secret پیکربندی شده و token دقیقاً برابرش باشد — بدون نیاز به DB (تابع خالص).
func TestIsTestToken(t *testing.T) {
	cases := []struct {
		name   string
		secret string
		token  string
		want   bool
	}{
		{"empty secret disables bypass", "", "anything", false},
		{"empty token never matches", "s3cr3t", "", false},
		{"both empty", "", "", false},
		{"matching secret and token", "s3cr3t", "s3cr3t", true},
		{"mismatching token", "s3cr3t", "wrong", false},
		{"case-sensitive mismatch", "S3cr3t", "s3cr3t", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isTestToken(c.secret, c.token); got != c.want {
				t.Errorf("isTestToken(%q, %q) = %v, want %v", c.secret, c.token, got, c.want)
			}
		})
	}
}

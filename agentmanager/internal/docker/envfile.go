package docker

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// mergeEnv base و overlay را به یک env slice داکری ("K=V") تبدیل می‌کند.
// کلید تکراری: overlay (env مخصوص deploy) همیشه برنده است.
// خروجی sort شده تا deterministic باشد (برای تست و مقایسه‌ی docker inspect).
func mergeEnv(base, overlay map[string]string) []string {
	merged := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overlay {
		merged[k] = v
	}
	env := make([]string, 0, len(merged))
	for k, v := range merged {
		// فقط k=v — هیچ shell expansion نیست
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	return env
}

// mergeEnvMaps دو map را ادغام می‌کند و map جدیدی برمی‌گرداند.
// overlay برنده است — برای ترکیب چند لایه قبل از تبدیل به []string.
func mergeEnvMaps(base, overlay map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

// parseEnvFileIfExists فایل KEY=VALUE را می‌خواند؛ اگر وجود نداشت map خالی برمی‌گرداند.
func parseEnvFileIfExists(path string) (map[string]string, error) {
	m, err := ParseEnvFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	return m, err
}

// ParseEnvFile یک فایل KEY=VALUE ساده را می‌خواند (فرمت .env: خط خالی و
// خطِ با # نادیده گرفته می‌شود). عمداً هیچ expansion یا quote-پردازشی
// انجام نمی‌دهد — مقدار همان چیزی است که بعد از اولین '=' آمده.
func ParseEnvFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			return nil, fmt.Errorf("%s:%d: خط نامعتبر (فرمت KEY=VALUE لازم است): %q", path, i+1, line)
		}
		out[k] = v
	}
	return out, nil
}

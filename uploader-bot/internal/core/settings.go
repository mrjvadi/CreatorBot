package core

import (
	"context"
	"strconv"
)

// GetSetting مقدار یک تنظیم را برمی‌گرداند و اگر خالی بود، پیش‌فرض را می‌دهد.
func (a *App) GetSetting(ctx context.Context, key, def string) string {
	v := a.Store.GetSetting(ctx, key)
	if v == "" {
		return def
	}
	return v
}

// GetSettingInt مقدار عددی یک تنظیم را برمی‌گرداند (با پیش‌فرض در صورت خطا).
func (a *App) GetSettingInt(ctx context.Context, key string, def int) int {
	v := a.GetSetting(ctx, key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

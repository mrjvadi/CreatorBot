package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Step وضعیت گفتگوی جاری کاربر (ماشین حالت ساده مبتنی بر Redis).
type Step string

// UserState وضعیت ذخیره‌شده‌ی کاربر: مرحله‌ی جاری + داده‌های همراه.
type UserState struct {
	Step Step              `json:"s"`
	Data map[string]string `json:"d,omitempty"`
}

func (a *App) stateKey(uid int64) string {
	return fmt.Sprintf("upl:state:%s:%d", a.InstanceID, uid)
}

// GetState وضعیت جاری کاربر را از کش می‌خواند.
func (a *App) GetState(ctx context.Context, uid int64) UserState {
	if a.Cache == nil {
		return UserState{}
	}
	raw, err := a.Cache.Get(ctx, a.stateKey(uid))
	a.LogErr("GetState", err)
	if raw == "" {
		return UserState{}
	}
	var st UserState
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		a.LogErr("GetState: unmarshal", err)
		return UserState{}
	}
	return st
}

// SetState وضعیت کامل کاربر را ذخیره می‌کند (TTL ۱۵ دقیقه).
func (a *App) SetState(ctx context.Context, uid int64, st UserState) {
	if a.Cache == nil {
		return
	}
	b, err := json.Marshal(st)
	if err != nil {
		a.LogErr("SetState: marshal", err)
		return
	}
	a.LogErr("SetState: cache set", a.Cache.Set(ctx, a.stateKey(uid), string(b), 15*time.Minute))
}

// SetStep فقط مرحله را تنظیم می‌کند (داده‌ها ریست می‌شوند).
func (a *App) SetStep(ctx context.Context, uid int64, s Step) {
	a.SetState(ctx, uid, UserState{Step: s, Data: make(map[string]string)})
}

// SetStepData مرحله را تنظیم و یک کلید/مقدار به داده‌ها اضافه می‌کند.
func (a *App) SetStepData(ctx context.Context, uid int64, s Step, key, val string) {
	st := a.GetState(ctx, uid)
	if st.Data == nil {
		st.Data = make(map[string]string)
	}
	st.Step = s
	st.Data[key] = val
	a.SetState(ctx, uid, st)
}

// ClearState وضعیت کاربر را پاک می‌کند.
func (a *App) ClearState(ctx context.Context, uid int64) {
	if a.Cache != nil {
		a.LogErr("ClearState", a.Cache.Del(ctx, a.stateKey(uid)))
	}
}

// ── بافر آلبوم (آپلود چند فایل به‌صورت media group) ──────────────

func (a *App) albumKey(uid int64) string {
	return fmt.Sprintf("upl:album:%s:%d", a.InstanceID, uid)
}

// AlbumAdd یک فایل را به‌صورت اتمیک به بافر آلبوم اضافه می‌کند.
// از LPush استفاده می‌شود تا در هجوم هم‌زمانِ آیتم‌های یک media group،
// هیچ فایلی به‌خاطر lost-update از دست نرود.
func (a *App) AlbumAdd(ctx context.Context, uid int64, fileID string) {
	if a.Cache == nil {
		return
	}
	a.LogErr("AlbumAdd", a.Cache.LPush(ctx, a.albumKey(uid), fileID))
}

// AlbumDrain همه‌ی فایل‌های بافر را به ترتیبِ ورود برمی‌گرداند و بافر را خالی می‌کند.
func (a *App) AlbumDrain(ctx context.Context, uid int64) []string {
	if a.Cache == nil {
		return nil
	}
	key := a.albumKey(uid)
	var ids []string
	for {
		// BLPop حداقل ۱ ثانیه تایم‌اوت می‌پذیرد؛ آیتم‌های موجود فوری برمی‌گردند
		// و فقط آخرین فراخوانیِ خالی ۱ ثانیه صبر می‌کند.
		vals, err := a.Cache.BLPop(ctx, time.Second, key)
		if err != nil || len(vals) < 2 {
			break
		}
		ids = append(ids, vals[1]) // vals[0]=key, vals[1]=value
	}
	// LPush معکوس می‌چیند؛ برای بازگرداندن ترتیب ورود، معکوس می‌کنیم.
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
	}
	return ids
}

// AlbumClear بافر آلبوم کاربر را پاک می‌کند.
func (a *App) AlbumClear(ctx context.Context, uid int64) {
	if a.Cache != nil {
		a.LogErr("AlbumClear", a.Cache.Del(ctx, a.albumKey(uid)))
	}
}

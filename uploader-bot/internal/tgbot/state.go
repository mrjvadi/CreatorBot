package tgbot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type step string

const (
	stepIdle step = ""

	// ادمین - ساخت کد
	stepCodeType    step = "code:type"
	stepCodeLimit   step = "code:limit"
	stepCodeExpiry  step = "code:expiry"
	stepCodeFiles   step = "code:files"   // جمع‌آوری فایل‌های آلبوم
	stepCodeConfirm step = "code:confirm"

	// ادمین - تنظیمات
	stepSettingKey   step = "setting:key"
	stepSettingValue step = "setting:value"

	// ادمین - broadcast
	stepBroadcast step = "broadcast:msg"
)

type state struct {
	Step step              `json:"s"`
	Data map[string]string `json:"d,omitempty"`
}

const ttl = 15 * time.Minute

func stateKey(uid int64) string { return fmt.Sprintf("ul:s:%d", uid) }

func (h *Handler) getState(ctx context.Context, uid int64) state {
	raw, err := h.eng.Cache.Get(ctx, stateKey(uid))
	if err != nil || raw == "" {
		return state{Step: stepIdle, Data: map[string]string{}}
	}
	var s state
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return state{Step: stepIdle, Data: map[string]string{}}
	}
	if s.Data == nil {
		s.Data = map[string]string{}
	}
	return s
}

func (h *Handler) setState(ctx context.Context, uid int64, s state) {
	data, _ := json.Marshal(s)
	h.eng.Cache.Set(ctx, stateKey(uid), string(data), ttl)
}

func (h *Handler) clearState(ctx context.Context, uid int64) {
	h.eng.Cache.Del(ctx, stateKey(uid))
}

func (h *Handler) setStep(ctx context.Context, uid int64, st step, kv ...string) {
	s := h.getState(ctx, uid)
	s.Step = st
	for i := 0; i+1 < len(kv); i += 2 {
		s.Data[kv[i]] = kv[i+1]
	}
	h.setState(ctx, uid, s)
}

// albumKey کلید cache برای جمع‌آوری فایل‌های آلبوم.
func albumKey(uid int64) string { return fmt.Sprintf("ul:album:%d", uid) }

type albumData struct {
	FileIDs []string `json:"ids"`
}

func (h *Handler) albumAdd(ctx context.Context, uid int64, fileID string) []string {
	raw, _ := h.eng.Cache.Get(ctx, albumKey(uid))
	var d albumData
	if raw != "" {
		json.Unmarshal([]byte(raw), &d)
	}
	d.FileIDs = append(d.FileIDs, fileID)
	data, _ := json.Marshal(d)
	h.eng.Cache.Set(ctx, albumKey(uid), string(data), ttl)
	return d.FileIDs
}

func (h *Handler) albumGet(ctx context.Context, uid int64) []string {
	raw, _ := h.eng.Cache.Get(ctx, albumKey(uid))
	var d albumData
	json.Unmarshal([]byte(raw), &d)
	return d.FileIDs
}

func (h *Handler) albumClear(ctx context.Context, uid int64) {
	h.eng.Cache.Del(ctx, albumKey(uid))
}

// suppress unused
var _ = ports.F

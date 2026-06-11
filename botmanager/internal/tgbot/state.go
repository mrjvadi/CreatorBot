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

	// سرور
	stepServerName step = "srv:name"
	stepServerIP   step = "srv:ip"

	// تمپلیت
	stepTmplType  step = "tmpl:type"
	stepTmplImage step = "tmpl:image"
	stepTmplTag   step = "tmpl:tag"
	stepTmplName  step = "tmpl:name"

	// لینک دعوت
	stepLinkType  step = "lnk:type"
	stepLinkLimit step = "lnk:limit"
	stepLinkLabel step = "lnk:label"

	// پلن
	stepPlanTmpl  step = "plan:tmpl"
	stepPlanName  step = "plan:name"
	stepPlanDays  step = "plan:days"
	stepPlanPrice step = "plan:price"

	// مدیریت کاربر
	stepUserAction step = "user:action"
	stepPlanSelect step = "plan:select"

	// wizard ساخت ربات
	stepWizardToken step = "wiz:token"
	stepLangSelect  step = "lang:select"

	// جستجو
	stepBotSearch  step = "bot:search"
	stepUserSearch step = "user:search"
)

type userState struct {
	Step step              `json:"s"`
	Data map[string]string `json:"d,omitempty"`
}

const stateTTL = 15 * time.Minute

func stateKey(uid int64) string {
	return fmt.Sprintf("bm:s:%d", uid)
}

func (h *Handler) getState(ctx context.Context, uid int64) userState {
	raw, err := h.cache.Get(ctx, stateKey(uid))
	if err != nil || raw == "" {
		return userState{Step: stepIdle, Data: map[string]string{}}
	}
	var s userState
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return userState{Step: stepIdle, Data: map[string]string{}}
	}
	if s.Data == nil {
		s.Data = map[string]string{}
	}
	return s
}

func (h *Handler) setState(ctx context.Context, uid int64, s userState) {
	data, _ := json.Marshal(s)
	h.cache.Set(ctx, stateKey(uid), string(data), stateTTL)
}

func (h *Handler) clearState(ctx context.Context, uid int64) {
	h.cache.Del(ctx, stateKey(uid))
}

func (h *Handler) setStep(ctx context.Context, uid int64, st step, kv ...string) {
	s := h.getState(ctx, uid)
	s.Step = st
	if s.Data == nil {
		s.Data = map[string]string{}
	}
	for i := 0; i+1 < len(kv); i += 2 {
		s.Data[kv[i]] = kv[i+1]
	}
	h.setState(ctx, uid, s)
}

// wizardPending توکن لینک در انتظار تأیید کاربر.
func (h *Handler) setWizardPending(ctx context.Context, uid int64, token string) {
	h.cache.Set(ctx, fmt.Sprintf("bm:wiz:%d", uid), token, 10*time.Minute)
}

func (h *Handler) getWizardPending(ctx context.Context, uid int64) string {
	val, _ := h.cache.Get(ctx, fmt.Sprintf("bm:wiz:%d", uid))
	return val
}

func (h *Handler) clearWizardPending(ctx context.Context, uid int64) {
	h.cache.Del(ctx, fmt.Sprintf("bm:wiz:%d", uid))
	_ = ports.F // suppress unused
}

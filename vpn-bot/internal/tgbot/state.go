package tgbot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type step string

const (
	stepIdle step = ""

	// خرید
	stepBuyPlan     step = "buy:plan"
	stepBuyPayment  step = "buy:payment"  // ورود رسید / hash پرداخت
	stepBuyReceipt  step = "buy:receipt"  // آپلود عکس رسید (card)

	// تمدید
	stepRenewSub    step = "renew:sub"
	stepRenewPayment step = "renew:payment"

	// ادمین — پنل
	stepAddPanelType step = "panel:type"
	stepAddPanelURL  step = "panel:url"
	stepAddPanelUser step = "panel:user"
	stepAddPanelPass step = "panel:pass"
	stepAddPanelCap  step = "panel:cap"

	// ادمین
	stepAdminPlanAdd  step = "admin:plan:add"
	stepAdminPanelAdd step = "admin:panel:add"
	stepAdminDiscount step = "admin:discount"
	stepAdminBroadcast step = "admin:broadcast"
)

type wizardState struct {
	Step step              `json:"s"`
	Data map[string]string `json:"d,omitempty"`
}

const stateTTL = 20 * time.Minute

func stateKey(uid int64) string { return fmt.Sprintf("vpn:s:%d", uid) }

func (h *Handler) getState(ctx context.Context, uid int64) wizardState {
	raw, err := h.cache.Get(ctx, stateKey(uid))
	if err != nil || raw == "" {
		return wizardState{Step: stepIdle, Data: map[string]string{}}
	}
	var s wizardState
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return wizardState{Step: stepIdle, Data: map[string]string{}}
	}
	if s.Data == nil {
		s.Data = map[string]string{}
	}
	return s
}

func (h *Handler) setState(ctx context.Context, uid int64, s wizardState) {
	data, _ := json.Marshal(s)
	h.cache.Set(ctx, stateKey(uid), string(data), stateTTL)
}

func (h *Handler) setStep(ctx context.Context, uid int64, st step, kv ...string) {
	s := h.getState(ctx, uid)
	s.Step = st
	for i := 0; i+1 < len(kv); i += 2 {
		s.Data[kv[i]] = kv[i+1]
	}
	h.setState(ctx, uid, s)
}

func (h *Handler) clearState(ctx context.Context, uid int64) {
	h.cache.Del(ctx, stateKey(uid))
}

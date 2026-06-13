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

	stepCampName    step = "camp:name"
	stepCampMedia   step = "camp:media"
	stepCampCaption step = "camp:caption"
	stepCampButton  step = "camp:button"
	stepCampURL     step = "camp:url"
	stepCampBudget  step = "camp:budget"
	stepCampCPJ     step = "camp:cpj"

	stepChannelFwd step = "channel:fwd"
	stepChannelCPJ step = "channel:cpj"

	stepAdminBroadcast step = "admin:broadcast"
	stepRejectNote     step = "admin:reject_note"
)

type wizardState struct {
	Step step              `json:"s"`
	Data map[string]string `json:"d,omitempty"`
}

const stateTTL = 20 * time.Minute

func stateKey(uid int64) string { return fmt.Sprintf("ads:s:%d", uid) }

func (h *Handler) getState(ctx context.Context, uid int64) wizardState {
	raw, _ := h.cache.Get(ctx, stateKey(uid))
	var s wizardState
	if raw != "" { json.Unmarshal([]byte(raw), &s) }
	if s.Data == nil { s.Data = map[string]string{} }
	return s
}

func (h *Handler) setState(ctx context.Context, uid int64, s wizardState) {
	data, _ := json.Marshal(s)
	h.cache.Set(ctx, stateKey(uid), string(data), stateTTL)
}

func (h *Handler) setStep(ctx context.Context, uid int64, st step, kv ...string) {
	s := h.getState(ctx, uid)
	s.Step = st
	for i := 0; i+1 < len(kv); i += 2 { s.Data[kv[i]] = kv[i+1] }
	h.setState(ctx, uid, s)
}

func (h *Handler) clearState(ctx context.Context, uid int64) {
	h.cache.Del(ctx, stateKey(uid))
}

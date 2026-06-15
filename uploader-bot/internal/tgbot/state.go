package tgbot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type step string

const (
	stepIdle        step = ""
	stepCodeFiles   step = "code:files"
	stepCodeCaption step = "code:caption"
	stepPassword    step = "password"
	stepSearch      step = "search"
	stepNewFolder   step = "folder:new"
	stepEditCaption step = "edit:caption"
	stepSetPassword step = "set:password"
	stepSetLimit    step = "set:limit"
	stepEditSetting step = "edit:setting"
	stepAddChannel  step = "channel:add"
	stepNewPlan     step = "plan:new"
	stepBroadcast   step = "broadcast"
	stepAddAdmin    step = "admin:add"
	stepSearchUser  step = "search:user"
)

type userState struct {
	Step step              `json:"s"`
	Data map[string]string `json:"d,omitempty"`
}

func (h *Handler) stateKey(uid int64) string {
	return fmt.Sprintf("upl:state:%s:%d", h.instanceID, uid)
}

func (h *Handler) getState(ctx context.Context, uid int64) userState {
	if h.cache == nil {
		return userState{}
	}
	raw, _ := h.cache.Get(ctx, h.stateKey(uid))
	if raw == "" {
		return userState{}
	}
	var st userState
	json.Unmarshal([]byte(raw), &st)
	return st
}

func (h *Handler) setState(ctx context.Context, uid int64, st userState) {
	if h.cache == nil {
		return
	}
	b, _ := json.Marshal(st)
	h.cache.Set(ctx, h.stateKey(uid), string(b), 15*time.Minute)
}

func (h *Handler) setStep(ctx context.Context, uid int64, s step) {
	h.setState(ctx, uid, userState{Step: s, Data: make(map[string]string)})
}

func (h *Handler) setStepData(ctx context.Context, uid int64, s step, key, val string) {
	st := h.getState(ctx, uid)
	if st.Data == nil {
		st.Data = make(map[string]string)
	}
	st.Step = s
	st.Data[key] = val
	h.setState(ctx, uid, st)
}

func (h *Handler) clearState(ctx context.Context, uid int64) {
	if h.cache != nil {
		h.cache.Del(ctx, h.stateKey(uid))
	}
}

// album buffer — برای آپلود چند فایل
func (h *Handler) albumKey(uid int64) string {
	return fmt.Sprintf("upl:album:%s:%d", h.instanceID, uid)
}

func (h *Handler) albumAdd(ctx context.Context, uid int64, fileID string) []string {
	raw, _ := h.cache.Get(ctx, h.albumKey(uid))
	var ids []string
	json.Unmarshal([]byte(raw), &ids)
	ids = append(ids, fileID)
	b, _ := json.Marshal(ids)
	h.cache.Set(ctx, h.albumKey(uid), string(b), 15*time.Minute)
	return ids
}

func (h *Handler) albumGet(ctx context.Context, uid int64) []string {
	raw, _ := h.cache.Get(ctx, h.albumKey(uid))
	var ids []string
	json.Unmarshal([]byte(raw), &ids)
	return ids
}

func (h *Handler) albumClear(ctx context.Context, uid int64) {
	if h.cache != nil {
		h.cache.Del(ctx, h.albumKey(uid))
	}
}

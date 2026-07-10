// state.go — state machine کاربر روی Redis (الگوی مشابه uploader-bot).
package tgbot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"
)

// step مرحله‌ی فعلی کاربر در یک wizard.
type step string

const (
	stepIdle step = ""

	// کانال و برچسب
	stepChannelAdd step = "channel:add" // انتظار برای username کانال
	stepTagAdd     step = "tag:add"     // نام برچسب جدید

	// کمپین
	stepCampaignName         step = "campaign:name"     // نام کمپین
	stepCampaignSchedule     step = "campaign:schedule" // ۴ خط زمان‌بندی
	stepCampaignEditName     step = "campaign:ename"    // ویرایش نام
	stepCampaignEditSchedule step = "campaign:esched"   // ویرایش زمان‌بندی

	// تبلیغ (اصلی + ریپلی‌ها)
	stepAdName         step = "ad:name"
	stepAdMain         step = "ad:main"
	stepAdReplies      step = "ad:replies"
	stepAdReplyMinutes step = "ad:reply_minutes" // مدت‌زمان (دقیقه) آخرین ریپلیِ دریافت‌شده
	stepAdEditName     step = "ad:edit_name"
	stepAdEditMain     step = "ad:edit_main"

	// قالب
	stepTemplateName step = "template:name"

	// تنظیمات
	stepSettingsReminder step = "settings:reminder"
)

// userState وضعیت یک کاربر در state machine؛ به‌صورت JSON در Redis ذخیره می‌شود.
type userState struct {
	Step step              `json:"s"`
	Data map[string]string `json:"d,omitempty"`
}

// stateKey کلید Redis برای state کاربر (per-instance).
func (h *Handler) stateKey(uid int64) string {
	return fmt.Sprintf("admanager:state:%s:%d", h.instanceID, uid)
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
	_ = json.Unmarshal([]byte(raw), &st)
	return st
}

func (h *Handler) setState(ctx context.Context, uid int64, st userState) {
	if h.cache == nil {
		return
	}
	b, _ := json.Marshal(st)
	_ = h.cache.Set(ctx, h.stateKey(uid), string(b), 15*time.Minute)
}

// setStep یک مرحله‌ی جدید بدون داده شروع می‌کند.
func (h *Handler) setStep(ctx context.Context, uid int64, s step) {
	h.setState(ctx, uid, userState{Step: s, Data: map[string]string{}})
}

// setStepData مرحله را تغییر می‌دهد و یک کلید داده را ذخیره می‌کند.
func (h *Handler) setStepData(ctx context.Context, uid int64, s step, key, val string) {
	st := h.getState(ctx, uid)
	if st.Data == nil {
		st.Data = map[string]string{}
	}
	st.Step = s
	st.Data[key] = val
	h.setState(ctx, uid, st)
}

func (h *Handler) clearState(ctx context.Context, uid int64) {
	if h.cache != nil {
		_ = h.cache.Del(ctx, h.stateKey(uid))
	}
}

// ── context (شناسه‌ی والدِ در حال ویرایش) ─────────────────────────
//
// callback_data تلگرام حداکثر ۶۴ بایت است؛ بنابراین در toggleها به‌جای
// جاسازی دو UUID، شناسه‌ی والد (کمپین/کانال) را این‌جا نگه می‌داریم و فقط
// شناسه‌ی هدف در دکمه می‌آید.

func (h *Handler) ctxKey(uid int64, kind string) string {
	return fmt.Sprintf("admanager:ctx:%s:%s:%d", h.instanceID, kind, uid)
}

func (h *Handler) setCtx(ctx context.Context, uid int64, kind, val string) {
	if h.cache != nil {
		_ = h.cache.Set(ctx, h.ctxKey(uid, kind), val, 15*time.Minute)
	}
}

func (h *Handler) getCtx(ctx context.Context, uid int64, kind string) string {
	if h.cache == nil {
		return ""
	}
	v, _ := h.cache.Get(ctx, h.ctxKey(uid, kind))
	return v
}

// handleStep ورودی متنی را بر اساس مرحله‌ی فعلی پردازش می‌کند.
//
// فقط ادمین می‌تواند در state machine باشد؛ بنابراین فرض بر این است که
// onText قبلاً دسترسی را بررسی کرده. مراحل کانال/تبلیغ در فازهای بعدی
// کامل می‌شوند.
func (h *Handler) handleStep(ctx context.Context, c tele.Context, st userState, text string) error {
	switch st.Step {

	// کانال و برچسب
	case stepChannelAdd:
		return h.handleChannelAdd(ctx, c, text)
	case stepTagAdd:
		return h.handleTagAdd(ctx, c, text)

	// کمپین
	case stepCampaignName:
		return h.handleCampaignName(ctx, c, st, text)
	case stepCampaignSchedule:
		return h.handleCampaignSchedule(ctx, c, st, text)
	case stepCampaignEditName:
		return h.handleCampaignEditName(ctx, c, st, text)
	case stepCampaignEditSchedule:
		return h.handleCampaignEditSchedule(ctx, c, st, text)

	// تبلیغ
	case stepAdName:
		return h.handleAdName(ctx, c, st, text)
	case stepAdMain:
		return h.handleAdMain(ctx, c, st)
	case stepAdReplies:
		return h.handleAdReplies(ctx, c, st)
	case stepAdReplyMinutes:
		return h.handleAdReplyMinutes(ctx, c, st, text)
	case stepAdEditName:
		return h.handleAdEditName(ctx, c, st, text)
	case stepAdEditMain:
		return h.handleAdEditMain(ctx, c, st)

	// قالب
	case stepTemplateName:
		return h.handleTemplateName(ctx, c, text)

	// تنظیمات
	case stepSettingsReminder:
		return h.handleSettingsReminder(ctx, c, text)

	default:
		h.clearState(ctx, c.Sender().ID)
		return c.Send("منوی اصلی:", kbAdminMain())
	}
}

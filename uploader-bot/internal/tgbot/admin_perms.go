package tgbot

import (
	"context"
	"strconv"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// adminCan بررسی می‌کند ادمین جاری دسترسی مشخص را دارد (مالک همیشه دارد).
func (h *Handler) adminCan(ctx context.Context, c tele.Context, perm string) bool {
	uid := c.Sender().ID
	if uid == h.OwnerID {
		return true
	}
	a, err := h.Store.GetAdmin(ctx, uid)
	h.LogErr("adminCan", err)
	if a == nil {
		return false
	}
	return a.Has(perm)
}

// sectionPerm دسترسی موردنیاز هر بخش پنل را برمی‌گرداند ("" = آزاد).
func sectionPerm(sec string) string {
	switch sec {
	case "upload", "codes", "folders", "slide", "pending", "preview":
		return models.PermUpload
	case "users", "reset":
		return models.PermUsers
	case "stats", "bcstat":
		return models.PermStats
	case "fjoin":
		return models.PermLocks
	case "plans":
		return models.PermPlans
	case "ads", "set", "tools", "togglebot", "search":
		return models.PermSettings
	case "bc":
		return models.PermBroadcast
	case "backup":
		return models.PermBackup
	case "admins":
		return models.PermAdmins
	}
	return ""
}

// permLabel نام فارسی هر دسترسی.
func permLabel(p string) string {
	switch p {
	case models.PermUpload:
		return "آپلود و رسانه‌ها"
	case models.PermBroadcast:
		return "ارسال همگانی"
	case models.PermUsers:
		return "کاربران"
	case models.PermLocks:
		return "قفل‌ها"
	case models.PermSettings:
		return "تنظیمات"
	case models.PermAdmins:
		return "ادمین‌ها"
	case models.PermPlans:
		return "اشتراک/پرداخت"
	case models.PermStats:
		return "آمار"
	case models.PermBackup:
		return "بکاپ"
	}
	return p
}

// adminPermsMenu منوی تنظیم دسترسی‌های یک ادمین.
func (h *Handler) adminPermsMenu(ctx context.Context, c tele.Context, tgIDStr string) error {
	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		return c.Edit(msgInvalid)
	}
	a, err := h.Store.GetAdmin(ctx, tgID)
	h.LogErr("adminPermsMenu", err)
	if a == nil {
		return c.Edit("❌ ادمین یافت نشد.")
	}
	if a.IsOwner {
		return c.Edit("👑 این کاربر مالک است و همه‌ی دسترسی‌ها را دارد.", kbBackHome())
	}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range models.AllPerms() {
		icon := "🔴"
		if a.Has(p) {
			icon = "🟢"
		}
		rows = append(rows, kb.Row(kb.Data(icon+" "+permLabel(p), "aperm_t:"+tgIDStr+":"+p)))
	}
	rows = append(rows, kb.Row(kb.Data(btnBackLabel, "p:admins")))
	kb.Inline(rows...)
	return c.Edit("🔑 دسترسی‌های ادمین "+tgIDStr+":", kb)
}

// adminTogglePerm یک دسترسی ادمین را روشن/خاموش می‌کند.
func (h *Handler) adminTogglePerm(ctx context.Context, c tele.Context, tgIDStr, perm string) error {
	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		return c.Edit(msgInvalid)
	}
	a, err := h.Store.GetAdmin(ctx, tgID)
	h.LogErr("adminTogglePerm: get", err)
	if a == nil {
		return c.Edit("❌ ادمین یافت نشد.")
	}
	// toggle
	has := false
	out := a.Perms[:0]
	for _, p := range a.Perms {
		if p == perm {
			has = true
			continue // حذف
		}
		out = append(out, p)
	}
	if !has {
		out = append(out, perm)
	}
	if err := h.Store.SetAdminPerms(ctx, tgID, out); err != nil {
		h.LogErr("adminTogglePerm: set", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ ذخیره نشد، دوباره امتحان کنید"})
	}
	return h.adminPermsMenu(ctx, c, tgIDStr)
}

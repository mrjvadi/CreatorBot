package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// ── Setting helpers ───────────────────────────────────────────

//func (h *Handler) getSetting(ctx context.Context, key, def string) string {
//	v := h.store.GetSetting(ctx, key)
//	if v == "" {
//		return def
//	}
//	return v
//}
//
//func (h *Handler) getSettingInt(ctx context.Context, key string, def int) int {
//	v := h.getSetting(ctx, key, "")
//	if v == "" {
//		return def
//	}
//	n, err := strconv.Atoi(v)
//	if err != nil {
//		return def
//	}
//	return n
//}

// ── handleStep — مسیریابی state machine ──────────────────────

func (h *Handler) handleStep(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID

	switch st.Step {
	case stepPassword:
		return h.userCheckPassword(ctx, c, st, text)
	case stepSearch:
		h.clearState(ctx, uid)
		return h.userSearch(ctx, c, text)
	case stepNewFolder:
		h.clearState(ctx, uid)
		if err := h.store.CreateFolder(ctx, &models.Folder{Name: text, IsActive: true}); err != nil {
			return c.Send("❌ خطا در ساخت پوشه.", kbAdmin())
		}
		return c.Send("✅ پوشه ساخته شد.", kbAdmin())
	case stepEditCaption:
		return h.adminSaveCaption(ctx, c, st, text)
	case stepSetPassword:
		return h.adminSavePassword(ctx, c, st, text)
	case stepSetLimit:
		return h.adminSaveLimit(ctx, c, st, text)
	case stepEditSetting:
		return h.adminSaveSetting(ctx, c, st, text)
	case stepAddChannel:
		return h.adminSaveChannel(ctx, c, text)
	case stepNewPlan:
		return h.adminSavePlan(ctx, c, st, text)
	case stepBroadcast:
		return h.adminDoBroadcast(ctx, c, text)
	case stepAddAdmin:
		return h.adminSaveAdmin(ctx, c, text)
	case stepSearchUser:
		h.clearState(ctx, uid)
		return h.adminShowUser(ctx, c, text)
	}
	h.clearState(ctx, uid)
	return nil
}

// ══════════════════════════════════════════════════════════════
// متدهای کاربر
// ══════════════════════════════════════════════════════════════

func (h *Handler) userCheckPassword(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	codeID := st.Data["code_id"]
	id, err := uuid.Parse(codeID)
	if err != nil {
		h.clearState(ctx, uid)
		return c.Send("❌ خطا.", kbUser(h.showSearch(ctx)))
	}
	code, _ := h.store.FindCodeByID(ctx, id)
	if code == nil {
		h.clearState(ctx, uid)
		return c.Send("❌ کد یافت نشد.", kbUser(h.showSearch(ctx)))
	}
	if text != code.Password {
		return c.Send("❌ رمز اشتباه است. دوباره وارد کنید:")
	}
	h.clearState(ctx, uid)
	user, _ := h.store.GetUser(ctx, uid)
	return h.sendFiles(ctx, c, user, code)
}

func (h *Handler) userSearch(ctx context.Context, c tele.Context, query string) error {
	codes, _ := h.store.SearchCodes(ctx, query)
	if len(codes) == 0 {
		return c.Send("🔍 نتیجه‌ای یافت نشد.", kbUser(h.showSearch(ctx)))
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 %d نتیجه:\n\n", len(codes)))
	for _, code := range codes {
		cap := code.Caption
		if len(cap) > 40 {
			cap = cap[:40] + "…"
		}
		sb.WriteString(fmt.Sprintf("📄 %s\n/get_%s\n\n", cap, code.Code))
	}
	return c.Send(sb.String(), kbUser(h.showSearch(ctx)))
}

func (h *Handler) userBuySubPlan(ctx context.Context, c tele.Context, arg string) error {
	plans, _ := h.store.ListSubPlans(ctx)
	if len(plans) == 0 {
		return c.Edit("❌ پلنی موجود نیست.")
	}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("%s — %.0f تومان (%d روز)", p.Name, p.Price, p.Days)
		rows = append(rows, kb.Row(kb.Data(label, "sub_pay:"+p.ID.String())))
	}
	kb.Inline(rows...)
	return c.Edit("💎 یک پلن انتخاب کنید:", kb)
}

func (h *Handler) userPaySub(ctx context.Context, c tele.Context, planID, gateway string) error {
	id, err := uuid.Parse(planID)
	if err != nil {
		return c.Edit("❌ پلن نامعتبر.")
	}
	plans, _ := h.store.ListSubPlans(ctx)
	var plan *models.SubPlan
	for i := range plans {
		if plans[i].ID == id {
			plan = &plans[i]
			break
		}
	}
	if plan == nil {
		return c.Edit("❌ پلن یافت نشد.")
	}
	// روش‌های پرداخت
	if gateway == "" {
		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(kb.Data("💳 کارت به کارت", "sub_pay:"+planID+":card")),
			kb.Row(kb.Data("🌐 آنلاین (زرین‌پال)", "sub_pay:"+planID+":zarinpal")),
			kb.Row(kb.Data("💎 تون (TON)", "sub_pay:"+planID+":ton")),
		)
		return c.Edit(fmt.Sprintf("روش پرداخت برای «%s»:", plan.Name), kb)
	}
	// ثبت payment در حالت pending
	uid := c.Sender().ID
	user, _ := h.store.GetUser(ctx, uid)
	pay := &models.Payment{
		UserID: user.ID, PlanID: plan.ID,
		Gateway: models.PaymentGateway(gateway),
		Amount:  plan.Price, Status: models.PaymentPending,
	}
	h.store.CreatePayment(ctx, pay)
	return c.Edit("✅ درخواست پرداخت ثبت شد.\nپس از تأیید ادمین اشتراک فعال می‌شود.")
}

func (h *Handler) userOpenFolder(ctx context.Context, c tele.Context, folderID string) error {
	var parentID *uuid.UUID
	if folderID != "" && folderID != "root" {
		if id, err := uuid.Parse(folderID); err == nil {
			parentID = &id
		}
	}
	folders, _ := h.store.ListFolders(ctx, parentID)
	codes, _, _ := h.store.ListCodes(ctx, parentID, 1, 50)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, f := range folders {
		rows = append(rows, kb.Row(kb.Data("📁 "+f.Name, "folder_open:"+f.ID.String())))
	}
	for _, code := range codes {
		label := code.Caption
		if label == "" {
			label = code.Code
		}
		rows = append(rows, kb.Row(kb.Data("📄 "+label, "code_resend:"+code.Code)))
	}
	kb.Inline(rows...)
	return c.Edit("📂 محتوا:", kb)
}

// ══════════════════════════════════════════════════════════════
// متدهای ادمین
// ══════════════════════════════════════════════════════════════

func (h *Handler) adminOnText(ctx context.Context, c tele.Context, text string) error {
	switch text {
	case btnNewCode:
		h.setStep(ctx, c.Sender().ID, stepCodeFiles)
		return c.Send("📤 فایل‌ها را ارسال کنید. پس از پایان «✅ تمام شد» بزنید.", kbAlbumDone())
	case btnCodeList:
		return h.adminListCodes(ctx, c)
	case btnFolders:
		return h.adminListFolders(ctx, c)
	case btnUsers:
		h.setStep(ctx, c.Sender().ID, stepSearchUser)
		return c.Send("👤 آیدی عددی یا یوزرنیم کاربر را بفرستید:", kbCancelOnly())
	case btnStats:
		return h.adminShowStats(ctx, c)
	case btnSettings:
		return c.Send("⚙️ تنظیمات:", kbSettings(h.store.GetAllSettings(ctx)))
	case btnSubPlans:
		return h.adminListPlans(ctx, c)
	case btnChannels:
		return h.adminListChannels(ctx, c)
	case btnBroadcast:
		h.setStep(ctx, c.Sender().ID, stepBroadcast)
		return c.Send("📢 پیام همگانی را بفرستید:", kbCancelOnly())
	case btnBackup:
		return h.adminDoBackup(ctx, c)
	case btnAdmins:
		return h.adminListAdmins(ctx, c)
	}
	return nil
}

func (h *Handler) adminHandleMedia(ctx context.Context, c tele.Context, uid int64) error {
	st := h.getState(ctx, uid)
	if st.Step != stepCodeFiles {
		return nil
	}
	fi := extractFileInfo(c)
	if fi == nil {
		return c.Send("❌ نوع فایل پشتیبانی نمی‌شود.")
	}
	// ذخیره فایل در buffer
	ids := h.albumAdd(ctx, uid, fi.fileID)
	return c.Send(fmt.Sprintf("✅ %d فایل اضافه شد. ادامه دهید یا «✅ تمام شد».", len(ids)))
}

func (h *Handler) adminListCodes(ctx context.Context, c tele.Context) error {
	codes, total, _ := h.store.ListCodes(ctx, nil, 1, 20)
	if len(codes) == 0 {
		return c.Send("📭 هیچ رسانه‌ای ثبت نشده.", kbAdmin())
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 رسانه‌ها (کل: %d):\n\n", total))
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, code := range codes {
		label := code.Caption
		if label == "" {
			label = code.Code
		}
		rows = append(rows, kb.Row(kb.Data("⚙️ "+label, "admin_code_edit:"+code.ID.String())))
	}
	kb.Inline(rows...)
	return c.Send(sb.String(), kb)
}

func (h *Handler) adminEditCodeMenu(ctx context.Context, c tele.Context, codeID string) error {
	id, err := uuid.Parse(codeID)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	code, _ := h.store.FindCodeByID(ctx, id)
	if code == nil {
		return c.Edit("❌ یافت نشد.")
	}
	info := fmt.Sprintf("⚙️ تنظیمات رسانه\n\n📛 کد: %s\n📥 دانلود: %d", code.Code, code.UsedCount)
	return c.Edit(info, kbCodeSettings(codeID, code.ForwardLock, code.AutoDelete > 0, code.Password != "", code.DownloadLimit))
}

func (h *Handler) adminDeleteCode(ctx context.Context, c tele.Context, codeID string) error {
	id, err := uuid.Parse(codeID)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	if err := h.store.DeleteCode(ctx, id); err != nil {
		return c.Edit("❌ خطا در حذف.")
	}
	return c.Edit("🗑 رسانه حذف شد.")
}

func (h *Handler) adminToggleCodeProp(ctx context.Context, c tele.Context, codeID, prop string) error {
	id, err := uuid.Parse(codeID)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	code, _ := h.store.FindCodeByID(ctx, id)
	if code == nil {
		return c.Edit("❌ یافت نشد.")
	}
	switch prop {
	case "forward_lock":
		code.ForwardLock = !code.ForwardLock
	case "sub_required":
		code.SubRequired = !code.SubRequired
	case "channel_lock":
		code.ChannelLock = !code.ChannelLock
	}
	h.store.UpdateCode(ctx, code)
	return h.adminEditCodeMenu(ctx, c, codeID)
}

func (h *Handler) adminSetAutoDelete(ctx context.Context, c tele.Context, codeID, sec string) error {
	id, err := uuid.Parse(codeID)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	code, _ := h.store.FindCodeByID(ctx, id)
	if code == nil {
		return c.Edit("❌ یافت نشد.")
	}
	n, _ := strconv.Atoi(sec)
	code.AutoDelete = n
	h.store.UpdateCode(ctx, code)
	return h.adminEditCodeMenu(ctx, c, codeID)
}

func (h *Handler) adminSaveCaption(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	id, _ := uuid.Parse(st.Data["code_id"])
	code, _ := h.store.FindCodeByID(ctx, id)
	if code == nil {
		return c.Send("❌ یافت نشد.", kbAdmin())
	}
	code.Caption = text
	h.store.UpdateCode(ctx, code)
	return c.Send("✅ کپشن بروزرسانی شد.", kbAdmin())
}

func (h *Handler) adminSavePassword(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	id, _ := uuid.Parse(st.Data["code_id"])
	code, _ := h.store.FindCodeByID(ctx, id)
	if code == nil {
		return c.Send("❌ یافت نشد.", kbAdmin())
	}
	if text == "0" || text == "حذف" {
		code.Password = ""
	} else {
		code.Password = text
	}
	h.store.UpdateCode(ctx, code)
	return c.Send("✅ رمز بروزرسانی شد.", kbAdmin())
}

func (h *Handler) adminSaveLimit(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	id, _ := uuid.Parse(st.Data["code_id"])
	code, _ := h.store.FindCodeByID(ctx, id)
	if code == nil {
		return c.Send("❌ یافت نشد.", kbAdmin())
	}
	n, _ := strconv.Atoi(text)
	code.DownloadLimit = n
	h.store.UpdateCode(ctx, code)
	return c.Send("✅ محدودیت دانلود تنظیم شد.", kbAdmin())
}

func (h *Handler) adminListFolders(ctx context.Context, c tele.Context) error {
	folders, _ := h.store.ListFolders(ctx, nil)
	items := make([]folderItem, 0, len(folders))
	for _, f := range folders {
		items = append(items, folderItem{ID: f.ID.String(), Name: f.Name})
	}
	return c.Send("📁 پوشه‌ها:", kbFolderList(items))
}

func (h *Handler) adminFolderOpen(ctx context.Context, c tele.Context, folderID string) error {
	id, err := uuid.Parse(folderID)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	codes, _, _ := h.store.ListCodes(ctx, &id, 1, 50)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📁 محتوای پوشه (%d رسانه):\n\n", len(codes)))
	for _, code := range codes {
		sb.WriteString(fmt.Sprintf("📄 %s\n", code.Code))
	}
	return c.Edit(sb.String())
}

func (h *Handler) adminFolderDelete(ctx context.Context, c tele.Context, folderID string) error {
	id, err := uuid.Parse(folderID)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	h.store.DeleteFolder(ctx, id)
	return c.Edit("🗑 پوشه حذف شد.")
}

func (h *Handler) adminForceJoinDelete(ctx context.Context, c tele.Context, chID string) error {
	id, err := uuid.Parse(chID)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	h.store.RemoveForceJoinChannel(ctx, id)
	return c.Edit("🗑 کانال حذف شد.")
}

func (h *Handler) adminListChannels(ctx context.Context, c tele.Context) error {
	channels, _ := h.store.ListForceJoinChannels(ctx)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ch := range channels {
		rows = append(rows, kb.Row(kb.Data("🗑 "+ch.Title, "admin_ch_del:"+ch.ID.String())))
	}
	kb.Inline(rows...)
	if len(channels) == 0 {
		return c.Send("📭 کانال جوین اجباری ثبت نشده.\nبرای افزودن، آیدی کانال را بفرستید.", kbAdmin())
	}
	return c.Send("📡 کانال‌های جوین اجباری:", kb)
}

func (h *Handler) adminSaveChannel(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	ch := &models.ForceJoinChannel{Title: text, Username: strings.TrimPrefix(text, "@"), IsActive: true}
	h.store.AddForceJoinChannel(ctx, ch)
	return c.Send("✅ کانال اضافه شد.", kbAdmin())
}

func (h *Handler) adminListPlans(ctx context.Context, c tele.Context) error {
	plans, _ := h.store.ListSubPlans(ctx)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("🗑 %s (%.0f ت)", p.Name, p.Price)
		rows = append(rows, kb.Row(kb.Data(label, "admin_sub_del:"+p.ID.String())))
	}
	kb.Inline(rows...)
	return c.Send("💎 پلن‌های اشتراک:", kb)
}

func (h *Handler) adminSubPlanDelete(ctx context.Context, c tele.Context, planID string) error {
	id, err := uuid.Parse(planID)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	h.store.DeleteSubPlan(ctx, id)
	return c.Edit("🗑 پلن حذف شد.")
}

func (h *Handler) adminSavePlan(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	// فرمت: نام:قیمت:روز
	parts := strings.Split(text, ":")
	if len(parts) != 3 {
		return c.Send("❌ فرمت: نام:قیمت:روز")
	}
	price, _ := strconv.ParseFloat(parts[1], 64)
	days, _ := strconv.Atoi(parts[2])
	h.clearState(ctx, uid)
	h.store.CreateSubPlan(ctx, &models.SubPlan{
		Name: parts[0], Price: price, Days: days, IsActive: true,
	})
	return c.Send("✅ پلن اضافه شد.", kbAdmin())
}

func (h *Handler) adminShowStats(ctx context.Context, c tele.Context) error {
	s := h.store.GetStats(ctx)
	msg := fmt.Sprintf(
		"📊 آمار ربات\n\n👥 کاربران: %d\n📄 رسانه‌ها: %d\n📁 فایل‌ها: %d\n🆕 امروز: %d\n💎 اشتراک فعال: %d",
		s.TotalUsers, s.TotalCodes, s.TotalFiles, s.TodayUsers, s.ActiveSubs,
	)
	return c.Send(msg, kbAdmin())
}

func (h *Handler) adminSaveSetting(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	key := st.Data["key"]
	h.store.SetSetting(ctx, key, text)
	return c.Send("✅ تنظیم ذخیره شد.", kbAdmin())
}

func (h *Handler) adminShowUser(ctx context.Context, c tele.Context, query string) error {
	user, _ := h.store.SearchUser(ctx, query)
	if user == nil {
		return c.Send("❌ کاربر یافت نشد.", kbAdmin())
	}
	status := "✅ فعال"
	if user.IsBlocked {
		status = "🚫 مسدود"
	}
	msg := fmt.Sprintf("👤 کاربر\n\n🆔 %d\n📛 %s\nوضعیت: %s\n📥 دانلود رایگان: %d",
		user.TelegramID, user.FirstName, status, user.FreeDownloads)
	kb := &tele.ReplyMarkup{}
	if user.IsBlocked {
		kb.Inline(kb.Row(kb.Data("✅ رفع مسدودی", "admin_user_unblock:"+strconv.FormatInt(user.TelegramID, 10))))
	} else {
		kb.Inline(kb.Row(kb.Data("🚫 مسدود کردن", "admin_user_block:"+strconv.FormatInt(user.TelegramID, 10))))
	}
	return c.Send(msg, kb)
}

func (h *Handler) adminToggleBlock(ctx context.Context, c tele.Context, tgIDStr string, block bool) error {
	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	h.store.BlockUser(ctx, tgID, block)
	if block {
		return c.Edit("🚫 کاربر مسدود شد.")
	}
	return c.Edit("✅ مسدودی رفع شد.")
}

func (h *Handler) adminConfirmPayment(ctx context.Context, c tele.Context, payID string) error {
	id, err := uuid.Parse(payID)
	if err != nil {
		return c.Edit("❌ نامعتبر.")
	}
	h.store.ConfirmPayment(ctx, id)
	return c.Edit("✅ پرداخت تأیید و اشتراک فعال شد.")
}

func (h *Handler) adminRejectPayment(ctx context.Context, c tele.Context, payID string) error {
	return c.Edit("❌ پرداخت رد شد.")
}

func (h *Handler) adminDoBroadcast(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	users, _, _ := h.store.ListUsers(ctx, 1, 100000)
	sent, failed := 0, 0
	for _, u := range users {
		if u.IsBlocked {
			continue
		}
		if _, err := c.Bot().Send(&tele.User{ID: u.TelegramID}, text); err != nil {
			failed++
		} else {
			sent++
		}
	}
	return c.Send(fmt.Sprintf("📢 ارسال شد.\n✅ موفق: %d\n❌ ناموفق: %d", sent, failed), kbAdmin())
}

func (h *Handler) adminDoBackup(ctx context.Context, c tele.Context) error {
	s := h.store.GetStats(ctx)
	h.store.CreateBackup(ctx, &models.Backup{
		TotalCodes: int(s.TotalCodes), TotalFiles: int(s.TotalFiles),
		CreatedBy: c.Sender().ID,
	})
	return c.Send(fmt.Sprintf("💾 بکاپ ساخته شد.\n📄 %d رسانه | 📁 %d فایل", s.TotalCodes, s.TotalFiles), kbAdmin())
}

func (h *Handler) adminBackupRestore(ctx context.Context, c tele.Context, backupID string) error {
	return c.Edit("♻️ بازیابی بکاپ آغاز شد.")
}

func (h *Handler) adminListAdmins(ctx context.Context, c tele.Context) error {
	admins, _ := h.store.ListAdmins(ctx)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👑 ادمین‌ها (%d):\n\n", len(admins)))
	for _, a := range admins {
		sb.WriteString(fmt.Sprintf("• %d", a.TelegramID))
		if a.IsOwner {
			sb.WriteString(" (مالک)")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\nبرای افزودن، آیدی عددی ادمین را بفرستید.")
	h.setStep(ctx, c.Sender().ID, stepAddAdmin)
	return c.Send(sb.String(), kbCancelOnly())
}

func (h *Handler) adminSaveAdmin(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	tgID, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return c.Send("❌ آیدی باید عددی باشد.", kbAdmin())
	}
	h.store.AddAdmin(ctx, tgID, "")
	return c.Send("✅ ادمین اضافه شد.", kbAdmin())
}

// notifyAdminsReport گزارش کاربر روی یک رسانه را به ادمین‌ها اطلاع می‌دهد.
func (h *Handler) notifyAdminsReport(ctx context.Context, c tele.Context, codeStr string) {
	admins, _ := h.store.ListAdmins(ctx)
	msg := fmt.Sprintf("⚠️ گزارش رسانه\n\n📛 کد: %s\n👤 گزارش‌دهنده: %d", codeStr, c.Sender().ID)
	for _, a := range admins {
		c.Bot().Send(&tele.User{ID: a.TelegramID}, msg)
	}
	// اگر هیچ ادمینی نبود، به owner
	if len(admins) == 0 && h.ownerID != 0 {
		c.Bot().Send(&tele.User{ID: h.ownerID}, msg)
	}
}

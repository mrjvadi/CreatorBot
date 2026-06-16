package tgbot

//
//import (
//	"archive/zip"
//	"bytes"
//	"context"
//	"encoding/json"
//	"fmt"
//	"strconv"
//	"strings"
//	"time"
//
//	tele "gopkg.in/telebot.v4"
//
//	"github.com/google/uuid"
//	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
//	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
//)
//
//// ── isAdmin ───────────────────────────────────────────────────
//
//func (h *Handler) isAdmin(c tele.Context) bool {
//	if c.Sender().ID == h.ownerID {
//		return true
//	}
//	return h.store.IsAdmin(context.Background(), c.Sender().ID)
//}
//
//// ── آپلود رسانه جدید ─────────────────────────────────────────
//
//func (h *Handler) adminNewCode(c tele.Context) error {
//	ctx := context.Background()
//	uid := c.Sender().ID
//	h.setStep(ctx, uid, stepCodeFiles)
//	return c.Send(
//		"📤 <b>آپلود رسانه جدید</b>\n\nفایل(ها) را ارسال کنید.\nبرای آلبوم چند فایل ارسال کنید سپس «✅ تمام شد» بزنید.",
//		tele.ModeHTML, kbAlbumDone(),
//	)
//}
//
//func (h *Handler) onMedia(c tele.Context) error {
//	if !h.isAdmin(c) {
//		// کاربر عادی — اگه upload کاربر فعال باشه
//		if h.getSetting(context.Background(), models.SettingUserUpload, "false") == "true" {
//			return h.userUploadMedia(c)
//		}
//		return nil
//	}
//	ctx := context.Background()
//	uid := c.Sender().ID
//	st := h.getState(ctx, uid)
//
//	fi := extractFile(c)
//	if fi == nil {
//		return nil
//	}
//
//	if st.Step == stepCodeFiles {
//		ids := h.albumAdd(ctx, uid, fi.FileID+"|"+fi.FileType+"|"+fi.Caption+"|"+fi.Thumbnail)
//		return c.Send(
//			fmt.Sprintf("✅ فایل %d اضافه شد.\nبرای پایان «✅ تمام شد» بزنید.", len(ids)),
//			kbAlbumDone(),
//		)
//	}
//	return nil
//}
//
//type fileInfo struct {
//	FileID    string
//	FileType  string
//	Caption   string
//	Thumbnail string
//}
//
//func extractFile(c tele.Context) *fileInfo {
//	m := c.Message()
//	switch {
//	case m.Video != nil:
//		th := ""
//		if m.Video.Thumbnail != nil {
//			th = m.Video.Thumbnail.FileID
//		}
//		return &fileInfo{m.Video.FileID, "video", m.Caption, th}
//	case m.Photo != nil:
//		return &fileInfo{m.Photo.FileID, "photo", m.Caption, ""}
//	case m.Document != nil:
//		return &fileInfo{m.Document.FileID, "document", m.Caption, ""}
//	case m.Audio != nil:
//		return &fileInfo{m.Audio.FileID, "audio", m.Caption, ""}
//	case m.Animation != nil:
//		return &fileInfo{m.Animation.FileID, "animation", m.Caption, ""}
//	case m.Voice != nil:
//		return &fileInfo{m.Voice.FileID, "voice", m.Caption, ""}
//	}
//	return nil
//}
//
//func (h *Handler) adminFinishUpload(ctx context.Context, c tele.Context) error {
//	uid := c.Sender().ID
//	album := h.albumGet(ctx, uid)
//	h.albumClear(ctx, uid)
//	h.clearState(ctx, uid)
//
//	if len(album) == 0 {
//		return c.Send("❌ هیچ فایلی آپلود نشد.", kbAdmin())
//	}
//
//	// ساخت کد
//	code := &models.Code{
//		Code:    h.store.GenerateUniqueCode(ctx),
//		Type:    models.CodeUnlimited,
//		IsAlbum: len(album) > 1,
//	}
//	if err := h.store.CreateCode(ctx, code); err != nil {
//		return c.Send("❌ خطا در ذخیره کد.", kbAdmin())
//	}
//
//	// ذخیره فایل‌ها
//	for i, entry := range album {
//		parts := strings.SplitN(entry, "|", 4)
//		if len(parts) < 2 {
//			continue
//		}
//		f := &models.File{
//			FileID: parts[0], FileType: parts[1],
//		}
//		if len(parts) > 2 {
//			f.Caption = parts[2]
//		}
//		if len(parts) > 3 {
//			f.Thumbnail = parts[3]
//		}
//		h.store.CreateFile(ctx, f)
//		h.store.AddFileToCode(ctx, code.ID, f.ID, i)
//	}
//
//	return c.Send(
//		fmt.Sprintf(
//			"✅ <b>رسانه ذخیره شد!</b>\n\n🆔 کد: <code>%s</code>\n📦 فایل: %d عدد\n\nلینک: <code>https://t.me/%s?start=%s</code>",
//			code.Code, len(album), h.store.GetSetting(ctx, "bot_username"), code.Code,
//		),
//		tele.ModeHTML, kbAdmin(),
//	)
//}
//
//// ── لیست کدها ────────────────────────────────────────────────
//
//func (h *Handler) adminCodeList(c tele.Context) error {
//	return h.adminCodeListPage(c, nil, 1)
//}
//
//func (h *Handler) adminCodeListPage(c tele.Context, folderID *uuid.UUID, page int) error {
//	ctx := context.Background()
//	codes, total, err := h.store.ListCodes(ctx, folderID, page, 10)
//	if err != nil {
//		return c.Send("❌ خطا.", kbAdmin())
//	}
//
//	if len(codes) == 0 {
//		return c.Send("📋 هیچ رسانه‌ای موجود نیست.", kbAdmin())
//	}
//
//	var sb strings.Builder
//	sb.WriteString(fmt.Sprintf("📋 <b>رسانه‌ها</b> (کل: %d)\n\n", total))
//	for _, code := range codes {
//		sb.WriteString(fmt.Sprintf("📦 <code>%s</code>", code.Code))
//		if code.Caption != "" {
//			cap := code.Caption
//			if len(cap) > 40 {
//				cap = cap[:40] + "..."
//			}
//			sb.WriteString(" — " + cap)
//		}
//		sb.WriteString(fmt.Sprintf(" | %d بار\n", code.UsedCount))
//	}
//
//	kb := &tele.ReplyMarkup{}
//	var rows []tele.Row
//	for _, code := range codes {
//		rows = append(rows, kb.Row(
//			kb.Data("⚙️ "+code.Code, "code_settings:"+code.ID.String()),
//		))
//	}
//	// pagination
//	var navRow []tele.Btn
//	if page > 1 {
//		navRow = append(navRow, kb.Data("⬅️", fmt.Sprintf("code_page:%d", page-1)))
//	}
//	if int64(page*10) < total {
//		navRow = append(navRow, kb.Data("➡️", fmt.Sprintf("code_page:%d", page+1)))
//	}
//	if len(navRow) > 0 {
//		rows = append(rows, kb.Row(navRow...))
//	}
//	rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "admin_main")))
//	kb.Inline(rows...)
//
//	return c.Send(sb.String(), tele.ModeHTML, kb)
//}
//
//// ── تنظیمات کد ───────────────────────────────────────────────
//
//func (h *Handler) adminCodeSettings(ctx context.Context, c tele.Context, codeIDStr string) error {
//	id, err := uuid.Parse(codeIDStr)
//	if err != nil {
//		return c.Edit("❌ کد نامعتبر.")
//	}
//	code, err := h.store.FindCodeByID(ctx, id)
//	if err != nil || code == nil {
//		return c.Edit("❌ کد یافت نشد.")
//	}
//
//	info := fmt.Sprintf(
//		"⚙️ <b>تنظیمات رسانه</b>\n\n🆔 کد: <code>%s</code>\n📦 فایل‌ها: %d\n📥 استفاده: %d\n",
//		code.Code, len(code.Files), code.UsedCount,
//	)
//	return c.Edit(info, tele.ModeHTML,
//		kbCodeSettings(codeIDStr, code.ForwardLock, code.AutoDelete > 0,
//			code.Password != "", code.DownloadLimit))
//}
//
//func (h *Handler) adminCodeDelete(ctx context.Context, c tele.Context, codeIDStr string) error {
//	id, _ := uuid.Parse(codeIDStr)
//	if err := h.store.DeleteCode(ctx, id); err != nil {
//		return c.Edit("❌ خطا در حذف.")
//	}
//	return c.Edit("🗑 رسانه حذف شد.")
//}
//
//func (h *Handler) adminToggleForward(ctx context.Context, c tele.Context, codeIDStr string) error {
//	id, _ := uuid.Parse(codeIDStr)
//	code, _ := h.store.FindCodeByID(ctx, id)
//	if code == nil {
//		return c.Edit("❌ یافت نشد.")
//	}
//	code.ForwardLock = !code.ForwardLock
//	h.store.UpdateCode(ctx, code)
//	return h.adminCodeSettings(ctx, c, codeIDStr)
//}
//
//func (h *Handler) adminToggleAntiDelete(ctx context.Context, c tele.Context, codeIDStr string) error {
//	id, _ := uuid.Parse(codeIDStr)
//	code, _ := h.store.FindCodeByID(ctx, id)
//	if code == nil {
//		return c.Edit("❌ یافت نشد.")
//	}
//	if code.AutoDelete > 0 {
//		code.AutoDelete = 0
//	} else {
//		code.AutoDelete = h.getSettingInt(ctx, models.SettingAutoDeleteDefault, 30)
//	}
//	h.store.UpdateCode(ctx, code)
//	return h.adminCodeSettings(ctx, c, codeIDStr)
//}
//
//func (h *Handler) adminEditCaptionSave(ctx context.Context, c tele.Context, codeIDStr, caption string) error {
//	h.clearState(ctx, c.Sender().ID)
//	id, _ := uuid.Parse(codeIDStr)
//	code, _ := h.store.FindCodeByID(ctx, id)
//	if code == nil {
//		return c.Send("❌ یافت نشد.", kbAdmin())
//	}
//	code.Caption = caption
//	h.store.UpdateCode(ctx, code)
//	return c.Send("✅ کپشن بروزرسانی شد.", kbAdmin())
//}
//
//func (h *Handler) adminSetPasswordSave(ctx context.Context, c tele.Context, codeIDStr, password string) error {
//	h.clearState(ctx, c.Sender().ID)
//	id, _ := uuid.Parse(codeIDStr)
//	code, _ := h.store.FindCodeByID(ctx, id)
//	if code == nil {
//		return c.Send("❌ یافت نشد.", kbAdmin())
//	}
//	if password == "0" {
//		password = ""
//	}
//	code.Password = password
//	h.store.UpdateCode(ctx, code)
//	msg := "✅ رمز عبور حذف شد."
//	if password != "" {
//		msg = "✅ رمز عبور تنظیم شد."
//	}
//	return c.Send(msg, kbAdmin())
//}
//
//func (h *Handler) adminSetLimitSave(ctx context.Context, c tele.Context, codeIDStr, limitStr string) error {
//	h.clearState(ctx, c.Sender().ID)
//	id, _ := uuid.Parse(codeIDStr)
//	code, _ := h.store.FindCodeByID(ctx, id)
//	if code == nil {
//		return c.Send("❌ یافت نشد.", kbAdmin())
//	}
//	limit, _ := strconv.Atoi(limitStr)
//	code.DownloadLimit = limit
//	h.store.UpdateCode(ctx, code)
//	return c.Send(fmt.Sprintf("✅ محدودیت دانلود: %d بار", limit), kbAdmin())
//}
//
//func (h *Handler) adminSendPreview(ctx context.Context, c tele.Context, codeIDStr string) error {
//	id, _ := uuid.Parse(codeIDStr)
//	code, _ := h.store.FindCodeByID(ctx, id)
//	if code == nil {
//		return c.Edit("❌ یافت نشد.")
//	}
//	channels, _ := h.store.ListPreviewChannels(ctx)
//	if len(channels) == 0 {
//		return c.Edit("❌ کانال پیش‌نمایش ثبت نشده.")
//	}
//	files, _ := h.store.GetFilesForCode(ctx, code.ID)
//	sig := h.getSetting(ctx, models.SettingSignature, "")
//	for _, ch := range channels {
//		for _, f := range files {
//			sendFileWithSig(c, f, sig, false)
//			_ = ch
//		}
//	}
//	return c.Edit("✅ پیش‌نمایش ارسال شد.")
//}
//
//// ── پوشه‌ها ───────────────────────────────────────────────────
//
//func (h *Handler) adminFolderList(c tele.Context) error {
//	ctx := context.Background()
//	folders, _ := h.store.ListFolders(ctx, nil)
//
//	items := make([]folderItem, 0, len(folders))
//	for _, f := range folders {
//		items = append(items, folderItem{ID: f.ID.String(), Name: f.Name})
//	}
//
//	kb := kbFolderList(items)
//	return c.Send(fmt.Sprintf("📁 <b>پوشه‌ها</b> (%d پوشه)", len(folders)),
//		tele.ModeHTML, kb)
//}
//
//func (h *Handler) adminNewFolderSave(ctx context.Context, c tele.Context, name string) error {
//	h.clearState(ctx, c.Sender().ID)
//	if err := h.store.CreateFolder(ctx, &models.Folder{Name: name}); err != nil {
//		return c.Send("❌ خطا در ایجاد پوشه.", kbAdmin())
//	}
//	return c.Send("✅ پوشه «"+name+"» ایجاد شد.", kbAdmin())
//}
//
//func (h *Handler) adminFolderDelete(ctx context.Context, c tele.Context, idStr string) error {
//	id, _ := uuid.Parse(idStr)
//	h.store.DeleteFolder(ctx, id)
//	return c.Edit("🗑 پوشه حذف شد.")
//}
//
//// ── کانال‌ها ──────────────────────────────────────────────────
//
//func (h *Handler) adminChannelList(c tele.Context) error {
//	ctx := context.Background()
//	channels, _ := h.store.ListForceJoinChannels(ctx)
//
//	var sb strings.Builder
//	sb.WriteString(fmt.Sprintf("📡 <b>کانال‌های جوین اجباری</b> (%d)\n\n", len(channels)))
//
//	kb := &tele.ReplyMarkup{}
//	var rows []tele.Row
//	for _, ch := range channels {
//		sb.WriteString(fmt.Sprintf("• %s (`%d`)\n", ch.Title, ch.ChatID))
//		rows = append(rows, kb.Row(
//			kb.Data("🗑 "+ch.Title, "channel_delete:"+ch.ID.String()),
//		))
//	}
//	rows = append(rows, kb.Row(kb.Data("➕ افزودن کانال", "channel_add")))
//	rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "admin_main")))
//	kb.Inline(rows...)
//
//	return c.Send(sb.String(), tele.ModeHTML, kb)
//}
//
//func (h *Handler) adminAddChannelSave(ctx context.Context, c tele.Context, text string) error {
//	h.clearState(ctx, c.Sender().ID)
//	// text = chat_id یا @username
//	var chatID int64
//	if strings.HasPrefix(text, "@") {
//		chat, err := c.Bot().ChatByUsername(text)
//		if err != nil {
//			return c.Send("❌ کانال یافت نشد.", kbAdmin())
//		}
//		chatID = chat.ID
//		ch := &models.ForceJoinChannel{
//			ChatID: chatID, Title: chat.Title, Username: strings.TrimPrefix(text, "@"),
//		}
//		h.store.AddForceJoinChannel(ctx, ch)
//	} else {
//		fmt.Sscan(text, &chatID)
//		if chatID == 0 {
//			return c.Send("❌ فرمت نادرست. @username یا chat_id ارسال کنید.", kbAdmin())
//		}
//		h.store.AddForceJoinChannel(ctx, &models.ForceJoinChannel{
//			ChatID: chatID, Title: fmt.Sprintf("Channel %d", chatID),
//		})
//	}
//	return c.Send(fmt.Sprintf("✅ کانال %d اضافه شد.", chatID), kbAdmin())
//}
//
//func (h *Handler) adminChannelDelete(ctx context.Context, c tele.Context, idStr string) error {
//	id, _ := uuid.Parse(idStr)
//	h.store.RemoveForceJoinChannel(ctx, id)
//	return c.Edit("🗑 کانال حذف شد.")
//}
//
//// ── اشتراک ────────────────────────────────────────────────────
//
//func (h *Handler) adminSubPlans(c tele.Context) error {
//	ctx := context.Background()
//	plans, _ := h.store.ListSubPlans(ctx)
//
//	var sb strings.Builder
//	sb.WriteString(fmt.Sprintf("💎 <b>پلن‌های اشتراک</b> (%d)\n\n", len(plans)))
//
//	kb := &tele.ReplyMarkup{}
//	var rows []tele.Row
//	for _, p := range plans {
//		sb.WriteString(fmt.Sprintf("• %s — %.0f تومان / %d روز\n", p.Name, p.Price, p.Days))
//		rows = append(rows, kb.Row(kb.Data("🗑 "+p.Name, "plan_delete:"+p.ID.String())))
//	}
//	rows = append(rows, kb.Row(kb.Data("➕ پلن جدید", "plan_new")))
//	rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "admin_main")))
//	kb.Inline(rows...)
//
//	return c.Send(sb.String(), tele.ModeHTML, kb)
//}
//
//func (h *Handler) adminNewPlanStep(ctx context.Context, c tele.Context, st userState, text string) error {
//	uid := c.Sender().ID
//	data := st.Data
//	if data == nil {
//		data = map[string]string{}
//	}
//
//	switch {
//	case data["name"] == "":
//		data["name"] = text
//		h.setState(ctx, uid, userState{Step: stepNewPlan, Data: data})
//		return c.Send("💰 قیمت پلن را به تومان وارد کنید:")
//	case data["price"] == "":
//		data["price"] = text
//		h.setState(ctx, uid, userState{Step: stepNewPlan, Data: data})
//		return c.Send("📅 مدت پلن را به روز وارد کنید:")
//	default:
//		h.clearState(ctx, uid)
//		days, _ := strconv.Atoi(text)
//		price, _ := strconv.ParseFloat(data["price"], 64)
//		plan := &models.SubPlan{Name: data["name"], Price: price, Days: days}
//		if err := h.store.CreateSubPlan(ctx, plan); err != nil {
//			return c.Send("❌ خطا در ایجاد پلن.", kbAdmin())
//		}
//		return c.Send(fmt.Sprintf("✅ پلن «%s» ایجاد شد.", plan.Name), kbAdmin())
//	}
//}
//
//func (h *Handler) adminPlanDelete(ctx context.Context, c tele.Context, idStr string) error {
//	id, _ := uuid.Parse(idStr)
//	h.store.DeleteSubPlan(ctx, id)
//	return c.Edit("🗑 پلن حذف شد.")
//}
//
//func (h *Handler) userBuyPlan(ctx context.Context, c tele.Context, planIDStr string) error {
//	defer c.Respond()
//	id, _ := uuid.Parse(planIDStr)
//	plans, _ := h.store.ListSubPlans(ctx)
//	var plan *models.SubPlan
//	for i := range plans {
//		if plans[i].ID == id {
//			plan = &plans[i]
//			break
//		}
//	}
//	if plan == nil {
//		return c.Edit("❌ پلن یافت نشد.")
//	}
//
//	kb := &tele.ReplyMarkup{}
//	kb.Inline(
//		kb.Row(kb.Data("💳 کارت به کارت", fmt.Sprintf("pay_card:%s", planIDStr))),
//		kb.Row(kb.Data("💎 TON", fmt.Sprintf("pay_ton:%s", planIDStr))),
//		kb.Row(kb.Data("🔙 بازگشت", "plans")),
//	)
//	return c.Edit(
//		fmt.Sprintf("💎 <b>%s</b>\n💰 قیمت: %.0f تومان\n📅 مدت: %d روز\n\nروش پرداخت را انتخاب کنید:",
//			plan.Name, plan.Price, plan.Days),
//		tele.ModeHTML, kb,
//	)
//}
//
//// ── کاربران ───────────────────────────────────────────────────
//
//func (h *Handler) adminUsers(c tele.Context) error {
//	ctx := context.Background()
//	stats := h.store.GetStats(ctx)
//
//	kb := &tele.ReplyMarkup{}
//	kb.Inline(
//		kb.Row(kb.Data("🔍 جستجوی کاربر", "search_user")),
//		kb.Row(kb.Data("🔙 بازگشت", "admin_main")),
//	)
//	return c.Send(
//		fmt.Sprintf("👥 <b>کاربران</b>\n\nکل: %d\nامروز: %d\nاشتراک فعال: %d",
//			stats.TotalUsers, stats.TodayUsers, stats.ActiveSubs),
//		tele.ModeHTML, kb,
//	)
//}
//
//func (h *Handler) adminSearchUserResult(ctx context.Context, c tele.Context, query string) error {
//	h.clearState(ctx, c.Sender().ID)
//	user, err := h.store.SearchUser(ctx, query)
//	if err != nil || user == nil {
//		return c.Send("❌ کاربر یافت نشد.", kbAdmin())
//	}
//
//	sub := "ندارد"
//	if user.HasActiveSub() {
//		sub = "✅ فعال تا " + user.SubExpiresAt.Format("2006-01-02")
//	}
//
//	kb := &tele.ReplyMarkup{}
//	kb.Inline(
//		kb.Row(
//			kb.Data("🚫 مسدود", fmt.Sprintf("block_user:%d", user.TelegramID)),
//			kb.Data("💎 تغییر اشتراک", fmt.Sprintf("change_sub:%d", user.TelegramID)),
//		),
//		kb.Row(kb.Data("🔙 بازگشت", "admin_users")),
//	)
//
//	return c.Send(
//		fmt.Sprintf("👤 <b>اطلاعات کاربر</b>\n\n🆔 %d\n👤 %s @%s\n💎 اشتراک: %s\n📥 دانلودها: %d\n🚫 مسدود: %v",
//			user.TelegramID, user.FirstName, user.Username,
//			sub, user.FreeDownloads, user.IsBlocked),
//		tele.ModeHTML, kb,
//	)
//}
//
//// ── آمار ──────────────────────────────────────────────────────
//
//func (h *Handler) adminStats(c tele.Context) error {
//	ctx := context.Background()
//	stats := h.store.GetStats(ctx)
//	return c.Send(
//		fmt.Sprintf(
//			"📊 <b>آمار ربات</b>\n\n"+
//				"👥 کاربران: %d\n"+
//				"📤 رسانه‌ها: %d\n"+
//				"📁 فایل‌ها: %d\n"+
//				"👤 کاربر امروز: %d\n"+
//				"💎 اشتراک فعال: %d",
//			stats.TotalUsers, stats.TotalCodes,
//			stats.TotalFiles, stats.TodayUsers, stats.ActiveSubs,
//		),
//		tele.ModeHTML, kbAdmin(),
//	)
//}
//
//// ── تنظیمات ───────────────────────────────────────────────────
//
//func (h *Handler) adminSettings(c tele.Context) error {
//	ctx := context.Background()
//	settings := h.store.GetAllSettings(ctx)
//	return c.Send("⚙️ <b>تنظیمات</b>", tele.ModeHTML, kbSettings(settings))
//}
//
//func (h *Handler) adminToggleSetting(ctx context.Context, c tele.Context, key string) error {
//	current := h.getSetting(ctx, key, "false")
//	newVal := "true"
//	if current == "true" {
//		newVal = "false"
//	}
//	h.store.SetSetting(ctx, key, newVal)
//	settings := h.store.GetAllSettings(ctx)
//	return c.Edit("⚙️ <b>تنظیمات</b>", tele.ModeHTML, kbSettings(settings))
//}
//
//func (h *Handler) adminSetSettingSave(ctx context.Context, c tele.Context, key, value string) error {
//	h.clearState(ctx, c.Sender().ID)
//	h.store.SetSetting(ctx, key, value)
//	return c.Send("✅ تنظیم ذخیره شد.", kbAdmin())
//}
//
//// ── بکاپ ──────────────────────────────────────────────────────
//
//func (h *Handler) adminBackup(c tele.Context) error {
//	ctx := context.Background()
//	uid := c.Sender().ID
//
//	_ = c.Send("⏳ در حال ساخت بکاپ...")
//
//	// جمع‌آوری داده‌ها
//	codes, _, _ := h.store.ListCodes(ctx, nil, 1, 99999)
//	settings := h.store.GetAllSettings(ctx)
//
//	backupData := map[string]any{
//		"version":   "3.0",
//		"timestamp": time.Now().Format(time.RFC3339),
//		"codes":     codes,
//		"settings":  settings,
//	}
//
//	// ساخت ZIP
//	var buf bytes.Buffer
//	zw := zip.NewWriter(&buf)
//	w, _ := zw.Create("backup.json")
//	json.NewEncoder(w).Encode(backupData)
//	zw.Close()
//
//	// ارسال فایل
//	doc := &tele.Document{
//		File:     tele.FromReader(bytes.NewReader(buf.Bytes())),
//		FileName: fmt.Sprintf("backup_%s.zip", time.Now().Format("20060102_150405")),
//		Caption:  fmt.Sprintf("💾 بکاپ\n📦 رسانه‌ها: %d\n⏰ %s", len(codes), time.Now().Format("2006-01-02 15:04")),
//	}
//
//	msg, err := c.Bot().Send(c.Recipient(), doc)
//	if err != nil {
//		h.log.Error("backup send", ports.F("err", err))
//		return c.Send("❌ خطا در ارسال بکاپ.")
//	}
//
//	// ذخیره اطلاعات بکاپ
//	h.store.CreateBackup(ctx, &models.Backup{
//		FileID:     msg.Document.FileID,
//		TotalCodes: len(codes),
//		CreatedBy:  uid,
//	})
//
//	return nil
//}
//
//// ── ارسال همگانی ──────────────────────────────────────────────
//
//func (h *Handler) adminBroadcastStart(c tele.Context) error {
//	ctx := context.Background()
//	uid := c.Sender().ID
//	h.setStep(ctx, uid, stepBroadcast)
//	return c.Send(
//		"📢 <b>ارسال همگانی</b>\n\nپیام یا فایل مورد نظر را ارسال کنید:",
//		tele.ModeHTML, kbCancelOnly(),
//	)
//}
//
//func (h *Handler) adminBroadcastSend(ctx context.Context, c tele.Context, text string) error {
//	h.clearState(ctx, c.Sender().ID)
//	users, _, _ := h.store.ListUsers(ctx, 1, 99999)
//
//	sent, failed := 0, 0
//	for _, user := range users {
//		if user.IsBlocked {
//			continue
//		}
//		_, err := c.Bot().Send(&tele.User{ID: user.TelegramID}, text)
//		if err != nil {
//			failed++
//		} else {
//			sent++
//		}
//		time.Sleep(50 * time.Millisecond) // anti-spam
//	}
//
//	return c.Send(
//		fmt.Sprintf("✅ ارسال همگانی تمام شد\n✉️ موفق: %d\n❌ ناموفق: %d", sent, failed),
//		kbAdmin(),
//	)
//}
//
//// ── ادمین‌ها ──────────────────────────────────────────────────
//
//func (h *Handler) adminAdminList(c tele.Context) error {
//	ctx := context.Background()
//	admins, _ := h.store.ListAdmins(ctx)
//
//	var sb strings.Builder
//	sb.WriteString(fmt.Sprintf("👑 <b>ادمین‌ها</b> (%d)\n\n", len(admins)))
//	kb := &tele.ReplyMarkup{}
//	var rows []tele.Row
//
//	for _, a := range admins {
//		sb.WriteString(fmt.Sprintf("• @%s (%d)", a.Username, a.TelegramID))
//		if a.IsOwner {
//			sb.WriteString(" 👑")
//		}
//		sb.WriteString("\n")
//		if !a.IsOwner {
//			rows = append(rows, kb.Row(
//				kb.Data("🗑 حذف @"+a.Username,
//					fmt.Sprintf("remove_admin:%d", a.TelegramID)),
//			))
//		}
//	}
//
//	rows = append(rows, kb.Row(kb.Data("➕ افزودن ادمین", "add_admin")))
//	rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "admin_main")))
//	kb.Inline(rows...)
//
//	return c.Send(sb.String(), tele.ModeHTML, kb)
//}
//
//func (h *Handler) adminAddAdminSave(ctx context.Context, c tele.Context, text string) error {
//	h.clearState(ctx, c.Sender().ID)
//	var tgID int64
//	username := ""
//	if strings.HasPrefix(text, "@") {
//		username = strings.TrimPrefix(text, "@")
//	} else {
//		fmt.Sscan(text, &tgID)
//	}
//	if tgID == 0 && username == "" {
//		return c.Send("❌ فرمت نادرست.", kbAdmin())
//	}
//	h.store.AddAdmin(ctx, tgID, username)
//	return c.Send("✅ ادمین اضافه شد.", kbAdmin())
//}
//
//// ── User Upload ───────────────────────────────────────────────
//
//func (h *Handler) userUploadMedia(c tele.Context) error {
//	// کاربر می‌تواند فایل آپلود کند اگه تنظیم فعال باشه
//	ctx := context.Background()
//	uid := c.Sender().ID
//
//	// بررسی auto approve
//	autoApprove := h.getSetting(ctx, models.SettingAutoApproveFiles, "false") == "true"
//
//	fi := extractFile(c)
//	if fi == nil {
//		return nil
//	}
//
//	f := &models.File{
//		FileID: fi.FileID, FileType: fi.FileType, Caption: fi.Caption,
//	}
//	h.store.CreateFile(ctx, f)
//
//	if autoApprove {
//		code := &models.Code{
//			Code:       h.store.GenerateUniqueCode(ctx),
//			Type:       models.CodeUnlimited,
//			UploaderID: uid,
//		}
//		h.store.CreateCode(ctx, code)
//		h.store.AddFileToCode(ctx, code.ID, f.ID, 0)
//		return c.Send(fmt.Sprintf("✅ فایل آپلود شد.\n🆔 کد: <code>%s</code>", code.Code), tele.ModeHTML)
//	}
//
//	// منتظر تأیید ادمین
//	return c.Send("⏳ فایل شما آپلود شد و در انتظار تأیید ادمین است.")
//}

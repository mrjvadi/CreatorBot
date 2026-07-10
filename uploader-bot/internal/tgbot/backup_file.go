package tgbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// backupFile ساختار فایل بکاپ کامل (فایل‌محور).
type backupFile struct {
	Version   int               `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
	Codes     []models.Code     `json:"codes"`
	Files     []models.File     `json:"files"`
	Settings  map[string]string `json:"settings"`
}

// kbBackupMenu کیبورد بکاپ/ریستور.
func kbBackupMenu() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("⬇️ گرفتن بکاپ", "backup_export")),
		kb.Row(kb.Data("♻️ بازیابی از فایل", "backup_restore")),
		kb.Row(kb.Data(btnBackLabel, "p:home")),
	)
	return kb
}

// adminBackupMenu منوی بکاپ/ریستور.
func (h *Handler) adminBackupMenu(ctx context.Context, c tele.Context) error {
	return c.Edit("💾 <b>بکاپ و بازیابی</b>\n\nبکاپ شامل کدها، فایل‌ها و تنظیمات است.", tele.ModeHTML, kbBackupMenu())
}

// adminBackupExport یک فایل JSON کامل می‌سازد و برای ادمین می‌فرستد.
func (h *Handler) adminBackupExport(ctx context.Context, c tele.Context) error {
	codes, err := h.Store.ListAllCodes(ctx)
	if err != nil {
		h.LogErr("adminBackupExport: list codes", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در خواندن کدها برای بکاپ"})
	}
	files, err := h.Store.ListAllFiles(ctx)
	if err != nil {
		h.LogErr("adminBackupExport: list files", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در خواندن فایل‌ها برای بکاپ"})
	}
	settings := h.Store.GetAllSettings(ctx)

	data := backupFile{
		Version:   1,
		CreatedAt: time.Now(),
		Codes:     codes,
		Files:     files,
		Settings:  settings,
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در ساخت بکاپ"})
	}

	name := fmt.Sprintf("backup_%s.json", time.Now().Format("2006-01-02_1504"))
	doc := &tele.Document{File: tele.FromReader(bytes.NewReader(b)), FileName: name}
	if _, err := c.Bot().Send(c.Recipient(), doc); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ ارسال فایل ناموفق"})
	}
	if err := h.Store.CreateBackup(ctx, &models.Backup{
		FileSize: int64(len(b)), TotalCodes: len(codes), TotalFiles: len(files),
		CreatedBy: c.Sender().ID,
	}); err != nil {
		// فایل بکاپ با موفقیت برای ادمین ارسال شد؛ فقط ثبتِ متادیتای تاریخچه
		// شکست خورده — نباید پیام موفقیت را به کاربر false negative کنیم، ولی
		// باید در لاگ بماند تا از قلم نیفتد.
		h.LogErr("adminBackupExport: save metadata", err)
	}
	return c.Respond(&tele.CallbackResponse{Text: "✅ بکاپ ارسال شد"})
}

// adminBackupRestoreAsk از ادمین می‌خواهد فایل بکاپ را بفرستد.
func (h *Handler) adminBackupRestoreAsk(ctx context.Context, c tele.Context) error {
	h.SetStep(ctx, c.Sender().ID, stepRestore)
	return c.Send("♻️ فایل بکاپ (JSON) را بفرستید.\n⚠️ داده‌های فعلی جایگزین می‌شوند.", kbCancelOnly())
}

// adminBackupRestore فایل آپلودشده را می‌خواند و داده‌ها را بازیابی می‌کند.
// از onMedia هنگام step=stepRestore فراخوانی می‌شود.
func (h *Handler) adminBackupRestore(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	doc := c.Message().Document
	if doc == nil {
		return c.Send("❌ لطفاً فایل بکاپ (JSON) را بفرستید.", kbAdmin())
	}
	rc, err := c.Bot().File(&doc.File)
	if err != nil {
		return c.Send("❌ دانلود فایل ناموفق بود.", kbAdmin())
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return c.Send("❌ خواندن فایل ناموفق بود.", kbAdmin())
	}
	var data backupFile
	if err := json.Unmarshal(raw, &data); err != nil {
		return c.Send("❌ فایل بکاپ معتبر نیست.", kbAdmin())
	}

	// پاک‌سازی و درج مجدد — چون این عملیات مخرب (پاک‌کردن داده‌ی فعلی) است،
	// هر خطا باید هم لاگ شود و هم صادقانه در پیام نهایی به ادمین گزارش شود
	// (نه یک «✅ بازیابی شد» که شکست‌های جزئی را قایم کند).
	if err := h.Store.DeleteAllCodes(ctx); err != nil {
		h.LogErr("adminBackupRestore: clear", err)
		return c.Send("❌ پاک‌سازی داده‌ی قبلی ناموفق بود؛ برای جلوگیری از داده‌ی ناقص، بازیابی متوقف شد.", kbAdmin())
	}

	filesOK, filesFail := 0, 0
	for i := range data.Files {
		if err := h.Store.InsertFileRaw(ctx, &data.Files[i]); err != nil {
			h.LogErr("adminBackupRestore: insert file", err)
			filesFail++
			continue
		}
		filesOK++
	}
	codesOK, codesFail := 0, 0
	for i := range data.Codes {
		if err := h.Store.InsertCodeRaw(ctx, &data.Codes[i]); err != nil {
			h.LogErr("adminBackupRestore: insert code", err)
			codesFail++
			continue
		}
		codesOK++
	}
	settingsFail := 0
	for k, v := range data.Settings {
		if err := h.Store.SetSetting(ctx, k, v); err != nil {
			h.LogErr("adminBackupRestore: set setting", err)
			settingsFail++
		}
	}

	if filesFail == 0 && codesFail == 0 && settingsFail == 0 {
		return c.Send(fmt.Sprintf("✅ بازیابی کامل بود.\n📄 %d کد · 📁 %d فایل", codesOK, filesOK), kbAdmin())
	}
	return c.Send(fmt.Sprintf(
		"⚠️ بازیابی با چند خطا انجام شد (جزئیات در لاگ سرور):\n"+
			"📄 کد: %d موفق، %d ناموفق\n📁 فایل: %d موفق، %d ناموفق\n⚙️ تنظیمات ناموفق: %d",
		codesOK, codesFail, filesOK, filesFail, settingsFail), kbAdmin())
}

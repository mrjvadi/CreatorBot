package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/docstore"
	"github.com/mrjvadi/creatorbot/shared-core/documents"
	"github.com/mrjvadi/creatorbot/shared-core/engine"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Deps dependency های handler.
type Deps struct {
	Engine    *engine.Engine
	Sender    ports.BotSender
	OwnerID   int64
	ChannelID int64
}

// Handler handler اصلی ربات آپلودر.
type Handler struct {
	d         Deps
	codeStore *docstore.CodeStore
	fileStore *docstore.FileStore
}

func NewHandler(d Deps) *Handler {
	return &Handler{
		d:         d,
		codeStore: docstore.NewCodeStore(d.Engine.Mongo, d.Engine.InstanceID),
		fileStore: docstore.NewFileStore(d.Engine.Mongo, d.Engine.InstanceID),
	}
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start", h.handleStart)
	b.Handle("/upload", h.handleUploadPrompt)
	b.Handle("/newcode", h.handleNewCode)
	b.Handle("/stats", h.handleStats)
	b.Handle("/block", h.handleBlock)
	b.Handle(tele.OnDocument, h.handleMedia)
	b.Handle(tele.OnVideo, h.handleMedia)
	b.Handle(tele.OnAudio, h.handleMedia)
	b.Handle(tele.OnPhoto, h.handleMedia)
	b.Handle(tele.OnVoice, h.handleMedia)
	b.Handle(tele.OnAnimation, h.handleMedia)
	b.Handle(tele.OnText, h.handleText)
}

func (h *Handler) handleStart(c tele.Context) error {
	ctx := context.Background()

	// ثبت کاربر — مستقیم به MongoDB
	h.d.Engine.Users.Upsert(ctx, &documents.BotUser{
		TelegramID: c.Sender().ID,
		Username:   c.Sender().Username,
		FirstName:  c.Sender().FirstName,
	})
	h.d.Engine.Stats.IncrementDaily(ctx, "unique_users", 1)

	// بررسی عضویت — مستقیم به Telegram API
	if h.d.ChannelID != 0 {
		status, err := h.d.Sender.GetChatMember(ctx, h.d.ChannelID, c.Sender().ID)
		if err != nil || !status.IsActive() {
			text, _ := h.d.Engine.Settings.Get(ctx, "not_member_text")
			if text == "" {
				text = "⛔️ برای استفاده باید عضو کانال باشید."
			}
			return c.Send(text)
		}
	}

	text, _ := h.d.Engine.Settings.Get(ctx, "welcome_text")
	if text == "" {
		text = fmt.Sprintf("سلام %s! 👋\nکد دریافت فایل را ارسال کنید.", c.Sender().FirstName)
	}
	return c.Send(text)
}

func (h *Handler) handleUploadPrompt(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	return c.Send("فایل را ارسال کنید.")
}

func (h *Handler) handleMedia(c tele.Context) error {
	if !h.isAdmin(c) {
		return h.handleText(c)
	}
	ctx := context.Background()

	fileID, fileType := extractFile(c)
	if fileID == "" {
		return nil
	}

	file := &documents.File{
		TelegramFileID: fileID,
		FileType:       fileType,
		Caption:        c.Message().Caption,
		UploaderID:     c.Sender().ID,
	}
	// ذخیره مستقیم در MongoDB
	if err := h.fileStore.Create(ctx, file); err != nil {
		h.d.Engine.Log.Error("fileStore.Create", ports.F("err", err))
		return c.Send("❌ خطا در ذخیره فایل.")
	}

	return c.Send(fmt.Sprintf(
		"✅ فایل ذخیره شد.\nID: <code>%s</code>\n\nبرای ساخت کد:\n/newcode <code>%s</code>",
		file.ID.Hex(), file.ID.Hex(),
	), tele.ModeHTML)
}

// /newcode <fileID> [once|limited:N|unlimited]
func (h *Handler) handleNewCode(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()

	args := strings.Fields(c.Message().Payload)
	if len(args) == 0 {
		return c.Send("استفاده: /newcode <fileID> [once|limited:5|unlimited]")
	}

	codeType := documents.CodeOnce
	maxUse := 1
	if len(args) >= 2 {
		switch {
		case args[1] == "unlimited":
			codeType, maxUse = documents.CodeUnlimited, 0
		case strings.HasPrefix(args[1], "limited:"):
			codeType = documents.CodeLimited
			fmt.Sscanf(strings.TrimPrefix(args[1], "limited:"), "%d", &maxUse)
		}
	}

	code := generateCode()
	// ذخیره مستقیم در MongoDB
	if err := h.codeStore.Create(ctx, &documents.Code{
		Code:    code,
		Type:    codeType,
		MaxUse:  maxUse,
		FileIDs: []string{args[0]},
	}); err != nil {
		return c.Send("❌ خطا در ساخت کد.")
	}

	limitText := map[documents.CodeType]string{
		documents.CodeOnce:      "یک‌بار",
		documents.CodeUnlimited: "نامحدود",
		documents.CodeLimited:   fmt.Sprintf("%d بار", maxUse),
	}[codeType]

	return c.Send(fmt.Sprintf(
		"✅ کد ساخته شد!\n\nکد: <code>%s</code>\nنوع: %s",
		code, limitText,
	), tele.ModeHTML)
}

func (h *Handler) handleText(c tele.Context) error {
	ctx := context.Background()
	text := strings.TrimSpace(c.Text())
	if text == "" {
		return nil
	}

	// بررسی بلاک — مستقیم از MongoDB
	user, _ := h.d.Engine.Users.FindByTelegramID(ctx, c.Sender().ID)
	if user != nil && user.IsBlocked {
		return c.Send("⛔️ دسترسی شما محدود شده است.")
	}

	// بررسی عضویت
	if h.d.ChannelID != 0 {
		status, err := h.d.Sender.GetChatMember(ctx, h.d.ChannelID, c.Sender().ID)
		if err != nil || !status.IsActive() {
			return c.Send("⛔️ برای دریافت فایل باید عضو کانال باشید.")
		}
	}

	// پیدا کردن کد — مستقیم از MongoDB
	code, err := h.codeStore.FindByCode(ctx, text)
	if err != nil {
		return c.Send("❌ خطای سرور.")
	}
	if code == nil || !h.codeStore.IsValid(code) {
		notFound, _ := h.d.Engine.Settings.Get(ctx, "not_found_text")
		if notFound == "" {
			notFound = "❌ کد یافت نشد یا منقضی شده."
		}
		return c.Send(notFound)
	}

	// دریافت فایل‌ها — مستقیم از MongoDB
	files, err := h.fileStore.FindByIDs(ctx, code.FileIDs)
	if err != nil || len(files) == 0 {
		return c.Send("❌ فایل یافت نشد.")
	}

	// ارسال فایل‌ها — مستقیم به Telegram API
	for _, f := range files {
		sendFile(c, f)
	}

	// ثبت استفاده و آمار — مستقیم در MongoDB
	h.codeStore.IncrementUse(ctx, code.ID.Hex())
	h.d.Engine.Stats.IncrementDaily(ctx, "total_actions", 1)

	return nil
}

func (h *Handler) handleStats(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	total, _ := h.d.Engine.Users.Count(ctx)
	return c.Send(fmt.Sprintf("📊 آمار\n\nکاربران: %d\nربات ID: %d",
		total, h.d.Engine.BotID))
}

func (h *Handler) handleBlock(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	var id int64
	if _, err := fmt.Sscanf(c.Message().Payload, "%d", &id); err != nil || id == 0 {
		return c.Send("استفاده: /block <telegram_id>")
	}
	ctx := context.Background()
	h.d.Engine.Users.SetBlocked(ctx, id, true)
	return c.Send(fmt.Sprintf("✅ کاربر %d بلاک شد.", id))
}

func (h *Handler) isAdmin(c tele.Context) bool {
	return c.Sender().ID == h.d.OwnerID
}

package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/util"
)

// ── Force Join ────────────────────────────────────────────────

// checkForceJoin قفل‌های اجباریِ کانال/گروه را بررسی می‌کند و آن‌هایی که کاربر
// عضو نیست را برمی‌گرداند. قفل‌های اختیاری و ربات/لینک اینجا اجبار نمی‌شوند.
func (h *Handler) checkForceJoin(ctx context.Context, c tele.Context) []models.ForceJoinChannel {
	channels, err := h.Store.ListForceJoinChannels(ctx)
	h.LogErr("checkForceJoin", err)
	var notJoined []models.ForceJoinChannel

	for _, ch := range channels {
		if !ch.IsMandatory() {
			continue
		}
		// عضویت فقط برای کانال/گروه قابل بررسی است (نه ربات/لینک).
		if ch.Kind == models.LockBot || ch.Kind == models.LockLink {
			continue
		}
		var chat *tele.Chat
		if ch.ChatID != 0 {
			chat = &tele.Chat{ID: ch.ChatID}
		} else if ch.Username != "" {
			chat = &tele.Chat{Username: ch.Username}
		} else {
			continue
		}
		member, err := c.Bot().ChatMemberOf(chat, c.Sender())
		if err != nil || member == nil ||
			member.Role == tele.Kicked || member.Role == tele.Left {
			notJoined = append(notJoined, ch)
		}
	}
	return notJoined
}

// sendJoinRequest پیام درخواست عضویت را با قفل‌های اجباری (نشده) + اختیاری می‌سازد.
func (h *Handler) sendJoinRequest(c tele.Context, mandatory []models.ForceJoinChannel) error {
	ctx := context.Background()
	notMemberText := h.Store.GetSetting(ctx, models.SettingNotMemberText)
	if notMemberText == "" {
		notMemberText = "📢 <b>یه قدم مونده!</b>\nبرای دریافت فایل، اول عضو این‌ها شو 👇"
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i := range mandatory {
		l := mandatory[i]
		if u := l.LinkURL(); u != "" {
			rows = append(rows, kb.Row(kb.URL("🔒 "+lockTitle(&l), u)))
		}
	}
	// قفل‌های اختیاری (پیشنهادی) هم نمایش داده می‌شوند.
	all, err := h.Store.ListForceJoinChannels(ctx)
	h.LogErr("sendJoinRequest", err)
	for i := range all {
		l := all[i]
		if l.IsMandatory() {
			continue
		}
		if u := l.LinkURL(); u != "" {
			rows = append(rows, kb.Row(kb.URL("🔓 "+lockTitle(&l), u)))
		}
	}
	rows = append(rows, kb.Row(kb.Data("✅ عضو شدم", "check_join")))
	kb.Inline(rows...)

	return c.Send(notMemberText, tele.ModeHTML, kb)
}

// ── File Helpers ──────────────────────────────────────────────

type fileInfo struct {
	fileID   string
	fileType string
}

func extractFileInfo(c tele.Context) *fileInfo {
	msg := c.Message()
	switch {
	case msg.Photo != nil:
		return &fileInfo{msg.Photo.FileID, "photo"}
	case msg.Video != nil:
		return &fileInfo{msg.Video.FileID, "video"}
	case msg.Document != nil:
		return &fileInfo{msg.Document.FileID, "document"}
	case msg.Audio != nil:
		return &fileInfo{msg.Audio.FileID, "audio"}
	case msg.Animation != nil:
		return &fileInfo{msg.Animation.FileID, "animation"}
	case msg.Voice != nil:
		return &fileInfo{msg.Voice.FileID, "voice"}
	case msg.VideoNote != nil:
		return &fileInfo{msg.VideoNote.FileID, "video_note"}
	case msg.Sticker != nil:
		return &fileInfo{msg.Sticker.FileID, "sticker"}
	}
	return nil
}

// albumCompatible مشخص می‌کند آیا همه‌ی فایل‌ها می‌توانند در یک آلبوم تلگرام
// ارسال شوند. ویس/استیکر/ویدیو‌نوت در آلبوم پشتیبانی نمی‌شوند.
func albumCompatible(files []models.File) bool {
	for _, f := range files {
		switch f.FileType {
		case "photo", "video", "audio", "document", "animation":
			// ok
		default:
			return false
		}
	}
	return true
}

// setInputCaption کپشن یک آیتم آلبوم را تنظیم می‌کند.
func setInputCaption(inp tele.Inputtable, caption string) {
	switch m := inp.(type) {
	case *tele.Photo:
		m.Caption = caption
	case *tele.Video:
		m.Caption = caption
	case *tele.Audio:
		m.Caption = caption
	case *tele.Document:
		m.Caption = caption
	case *tele.Animation:
		m.Caption = caption
	}
}

func fileToInput(f models.File) tele.Inputtable {
	file := tele.File{FileID: f.FileID}
	switch f.FileType {
	case "photo":
		return &tele.Photo{File: file, Caption: f.Caption}
	case "video":
		v := &tele.Video{File: file, Caption: f.Caption}
		if f.Thumbnail != "" {
			v.Thumbnail = &tele.Photo{File: tele.File{FileID: f.Thumbnail}}
		}
		return v
	case "audio":
		return &tele.Audio{File: file, Caption: f.Caption}
	case "document":
		return &tele.Document{File: file, Caption: f.Caption}
	case "animation":
		return &tele.Animation{File: file, Caption: f.Caption}
	}
	return nil
}

// archiveToStorage پیام آپلودشده را به کانال ذخیره‌سازی (در صورت تنظیم) کپی می‌کند
// و مرجع (chatID, msgID) را برمی‌گرداند. این مرجع در برابر تغییر توکن پایدار است.
func (h *Handler) archiveToStorage(ctx context.Context, c tele.Context) (int64, int, bool) {
	raw := h.Store.GetSetting(ctx, models.SettingStorageChannel)
	if raw == "" {
		return 0, 0, false
	}
	chatID, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || chatID == 0 {
		return 0, 0, false
	}
	m, err := c.Bot().Copy(&tele.Chat{ID: chatID}, c.Message())
	if err != nil || m == nil {
		h.Log.Error("archiveToStorage", ports.F("err", err))
		return 0, 0, false
	}
	return chatID, m.ID, true
}

// sendFromStorage فایل را از کانال ذخیره‌سازی به گیرنده کپی می‌کند (fallback).
func (h *Handler) sendFromStorage(c tele.Context, f models.File, so *tele.SendOptions) (*tele.Message, error) {
	src := &tele.Message{ID: f.StorageMsgID, Chat: &tele.Chat{ID: f.StorageChatID}}
	if so != nil {
		return c.Bot().Copy(c.Recipient(), src, so)
	}
	return c.Bot().Copy(c.Recipient(), src)
}

// sendMedia یک فایل را با کپشن و قالب‌بندی (entities) و گزینه‌های ارسال می‌فرستد.
// اگر به‌خاطر قالب‌بندیِ نامعتبر خطا بدهد، یک‌بار به‌صورت متن ساده دوباره تلاش می‌کند
// تا کپشن/رسانه هرگز به‌خاطر فرمت از بین نرود.
func sendMedia(c tele.Context, f models.File, caption string, entities []models.Entity, so *tele.SendOptions) (*tele.Message, error) {
	if so == nil {
		so = &tele.SendOptions{}
	}
	// telebot برای کپشن رسانه، Entities را درست به caption_entities نگاشت نمی‌کند؛
	// پس entities را به HTML تبدیل و با ParseMode=HTML می‌فرستیم.
	if len(entities) > 0 {
		so.ParseMode = tele.ModeHTML
		so.Entities = nil
		if m, err := sendMediaWith(c, f, util.EntitiesToHTML(caption, entities), so); err == nil {
			return m, nil
		}
		// fallback: متن ساده (اگر HTML خطا داد)
	}
	plain := &tele.SendOptions{ReplyMarkup: so.ReplyMarkup, Protected: so.Protected}
	return sendMediaWith(c, f, caption, plain)
}

func sendMediaWith(c tele.Context, f models.File, caption string, so *tele.SendOptions) (*tele.Message, error) {
	file := tele.File{FileID: f.FileID}
	switch f.FileType {
	case "photo":
		return c.Bot().Send(c.Recipient(), &tele.Photo{File: file, Caption: caption}, so)
	case "video":
		v := &tele.Video{File: file, Caption: caption}
		if f.Thumbnail != "" {
			v.Thumbnail = &tele.Photo{File: tele.File{FileID: f.Thumbnail}}
		}
		return c.Bot().Send(c.Recipient(), v, so)
	case "audio":
		return c.Bot().Send(c.Recipient(), &tele.Audio{File: file, Caption: caption}, so)
	case "animation":
		return c.Bot().Send(c.Recipient(), &tele.Animation{File: file, Caption: caption}, so)
	case "voice":
		return c.Bot().Send(c.Recipient(), &tele.Voice{File: file, Caption: caption}, so)
	case "video_note":
		return c.Bot().Send(c.Recipient(), &tele.VideoNote{File: file}, so)
	case "sticker":
		return c.Bot().Send(c.Recipient(), &tele.Sticker{File: file}, so)
	default:
		return c.Bot().Send(c.Recipient(), &tele.Document{File: file, Caption: caption}, so)
	}
}

// reactReportRows ردیف‌های «رای‌ها (لایک/دیسلایک)» و «گزارش» را می‌سازد.
// این‌ها همیشه زیر خودِ فایل قرار می‌گیرند.
func reactReportRows(code *models.Code, showLikes, showReport bool, realLikes, realDislikes int64) []tele.Row {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	if showLikes {
		likes := int64(code.FakeLikes) + realLikes
		rows = append(rows, kb.Row(
			kb.Data(fmt.Sprintf("👍 %d", likes), "react_like:"+code.Code),
			kb.Data(fmt.Sprintf("👎 %d", realDislikes), "react_dislike:"+code.Code),
		))
	}
	if showReport {
		rows = append(rows, kb.Row(kb.Data("⚠️ گزارش", "report:"+code.Code)))
	}
	return rows
}

// resendRow ردیف دکمه‌ی «ارسال مجدد».
func resendRow(code *models.Code) tele.Row {
	kb := &tele.ReplyMarkup{}
	return kb.Row(kb.Data("🔄 ارسال مجدد", "code_resend:"+code.Code))
}

// sendToChat یک فایل را به یک چت دلخواه (مثلاً کانال پیش‌نمایش) ارسال می‌کند.
func sendToChat(bot *tele.Bot, chatID int64, f models.File, caption string) (*tele.Message, error) {
	to := &tele.Chat{ID: chatID}
	file := tele.File{FileID: f.FileID}
	if caption == "" {
		caption = f.Caption
	}
	switch f.FileType {
	case "photo":
		return bot.Send(to, &tele.Photo{File: file, Caption: caption}, tele.ModeHTML)
	case "video":
		return bot.Send(to, &tele.Video{File: file, Caption: caption}, tele.ModeHTML)
	case "audio":
		return bot.Send(to, &tele.Audio{File: file, Caption: caption}, tele.ModeHTML)
	case "animation":
		return bot.Send(to, &tele.Animation{File: file, Caption: caption}, tele.ModeHTML)
	case "voice":
		return bot.Send(to, &tele.Voice{File: file, Caption: caption}, tele.ModeHTML)
	case "video_note":
		return bot.Send(to, &tele.VideoNote{File: file})
	case "sticker":
		return bot.Send(to, &tele.Sticker{File: file})
	default:
		return bot.Send(to, &tele.Document{File: file, Caption: caption}, tele.ModeHTML)
	}
}

func fileTypeIcon(t string) string {
	icons := map[string]string{
		"photo": "🖼", "video": "🎬", "audio": "🎵",
		"document": "📄", "animation": "🎭", "voice": "🎤",
		"sticker": "😊",
	}
	if icon, ok := icons[t]; ok {
		return icon
	}
	return "📁"
}

// ── Backup/Restore ────────────────────────────────────────────

// BackupData ساختار داده بکاپ.
type BackupData struct {
	Version   int                       `json:"version"`
	CreatedAt time.Time                 `json:"created_at"`
	Codes     []models.Code             `json:"codes"`
	Settings  map[string]string         `json:"settings"`
	Channels  []models.ForceJoinChannel `json:"channels"`
}

// ── Format ────────────────────────────────────────────────────

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatTime(t time.Time) string {
	return t.Format("2006/01/02 15:04")
}

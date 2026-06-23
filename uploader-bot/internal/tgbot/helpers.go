package tgbot

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// ── Force Join ────────────────────────────────────────────────

// checkForceJoin بررسی می‌کند کاربر در کانال‌های اجباری عضو است.
// لیست کانال‌هایی که عضو نیست را برمی‌گرداند.
func (h *Handler) checkForceJoin(ctx context.Context, c tele.Context) []models.ForceJoinChannel {
	channels, _ := h.store.ListForceJoinChannels(ctx)
	var notJoined []models.ForceJoinChannel

	for _, ch := range channels {
		member, err := c.Bot().ChatMemberOf(
			&tele.Chat{ID: ch.ChatID},
			c.Sender(),
		)
		if err != nil || member == nil ||
			member.Role == tele.Kicked || member.Role == tele.Left {
			notJoined = append(notJoined, ch)
		}
	}
	return notJoined
}

func (h *Handler) sendJoinRequest(c tele.Context, channels []models.ForceJoinChannel) error {
	notMemberText := h.store.GetSetting(context.Background(), models.SettingNotMemberText)
	if notMemberText == "" {
		notMemberText = "⚠️ <b>برای دریافت فایل باید در کانال‌های زیر عضو شوید:</b>"
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ch := range channels {
		url := ch.InviteURL
		if ch.Username != "" {
			url = "https://t.me/" + ch.Username
		}
		rows = append(rows, kb.Row(kb.URL("📢 "+ch.Title, url)))
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
	case msg.Sticker != nil:
		return &fileInfo{msg.Sticker.FileID, "sticker"}
	}
	return nil
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

func sendSingleFile(c tele.Context, f models.File, caption string, opts ...any) (*tele.Message, error) {
	file := tele.File{FileID: f.FileID}
	if caption == "" {
		caption = f.Caption
	}

	sendOpts := append([]any{tele.ModeHTML}, opts...)

	switch f.FileType {
	case "photo":
		return c.Bot().Send(c.Recipient(), &tele.Photo{File: file, Caption: caption}, sendOpts...)
	case "video":
		v := &tele.Video{File: file, Caption: caption}
		if f.Thumbnail != "" {
			v.Thumbnail = &tele.Photo{File: tele.File{FileID: f.Thumbnail}}
		}
		return c.Bot().Send(c.Recipient(), v, sendOpts...)
	case "audio":
		return c.Bot().Send(c.Recipient(), &tele.Audio{File: file, Caption: caption}, sendOpts...)
	case "animation":
		return c.Bot().Send(c.Recipient(), &tele.Animation{File: file, Caption: caption}, sendOpts...)
	case "voice":
		return c.Bot().Send(c.Recipient(), &tele.Voice{File: file, Caption: caption}, sendOpts...)
	case "sticker":
		return c.Bot().Send(c.Recipient(), &tele.Sticker{File: file})
	default:
		return c.Bot().Send(c.Recipient(), &tele.Document{File: file, Caption: caption}, sendOpts...)
	}
}

// kbMediaButtons دکمه‌های زیر رسانه را می‌سازد: لایک/دیسلایک (با شمارنده fake)،
// گزارش، ارسال مجدد. بر اساس تنظیمات نمایش داده می‌شوند.
func kbMediaButtons(code *models.Code, showLikes, showReport bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row

	if showLikes {
		likes := code.FakeLikes
		rows = append(rows, kb.Row(
			kb.Data(fmt.Sprintf("👍 %d", likes), "react_like:"+code.Code),
			kb.Data("👎", "react_dislike:"+code.Code),
		))
	}

	var bottomRow []tele.Btn
	bottomRow = append(bottomRow, kb.Data("🔄 ارسال مجدد", "code_resend:"+code.Code))
	if showReport {
		bottomRow = append(bottomRow, kb.Data("⚠️ گزارش", "report:"+code.Code))
	}
	rows = append(rows, kb.Row(bottomRow...))

	kb.Inline(rows...)
	return kb
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

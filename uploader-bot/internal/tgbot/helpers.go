package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/documents"
)

// ── عضویت اجباری ──────────────────────────────────────────

// checkMembership بررسی می‌کند کاربر عضو کانال اجباری است.
// اگه عضو نبود پیام مناسب می‌فرستد و error برمی‌گرداند.
func (h *Handler) checkMembership(ctx context.Context, c tele.Context) error {
	if h.channelID == 0 {
		return nil // force_join غیرفعال
	}
	status, err := h.sender.GetChatMember(ctx, h.channelID, c.Sender().ID)
	if err != nil || !status.IsActive() {
		chUsername, _ := h.eng.Settings.Get(ctx, "channel_username")
		notMemberText := h.setting(ctx, "not_member_text",
			"⛔️ برای استفاده از ربات باید در کانال عضو باشید.")
		return c.Send(notMemberText,
			kbJoinChannel(chUsername, fmt.Sprintf("%d", h.channelID)))
	}
	return nil
}

// ── ارسال فایل ────────────────────────────────────────────

func sendFile(c tele.Context, f documents.File) error {
	file := tele.FromFileID(f.TelegramFileID)
	switch f.FileType {
	case "video":
		return c.Send(&tele.Video{File: file, Caption: f.Caption}, tele.ModeHTML)
	case "audio":
		return c.Send(&tele.Audio{File: file, Caption: f.Caption}, tele.ModeHTML)
	case "photo":
		return c.Send(&tele.Photo{File: file, Caption: f.Caption}, tele.ModeHTML)
	case "voice":
		return c.Send(&tele.Voice{File: file})
	case "animation":
		return c.Send(&tele.Animation{File: file, Caption: f.Caption}, tele.ModeHTML)
	case "video_note":
		return c.Send(&tele.VideoNote{File: file})
	default:
		return c.Send(&tele.Document{File: file, Caption: f.Caption}, tele.ModeHTML)
	}
}

// ── استخراج فایل از پیام ──────────────────────────────────

type fileInfo struct {
	ID       string
	Type     string
	Caption  string
}

func extractFile(c tele.Context) *fileInfo {
	m := c.Message()
	switch {
	case m.Document != nil:
		return &fileInfo{m.Document.FileID, "document", m.Caption}
	case m.Video != nil:
		return &fileInfo{m.Video.FileID, "video", m.Caption}
	case m.Audio != nil:
		return &fileInfo{m.Audio.FileID, "audio", m.Caption}
	case m.Voice != nil:
		return &fileInfo{m.Voice.FileID, "voice", ""}
	case m.Animation != nil:
		return &fileInfo{m.Animation.FileID, "animation", m.Caption}
	case m.VideoNote != nil:
		return &fileInfo{m.VideoNote.FileID, "video_note", ""}
	case m.Photo != nil:
		p := m.Photo
		return &fileInfo{p[len(p)-1].FileID, "photo", m.Caption}
	}
	return nil
}

// ── parse helpers ──────────────────────────────────────────

func parseCodeType(text string) documents.CodeType {
	switch text {
	case btnOnce:
		return documents.CodeOnce
	case btnLimited:
		return documents.CodeLimited
	case btnUnlimited:
		return documents.CodeUnlimited
	case btnExpiry:
		return documents.CodeExpiry
	}
	return ""
}

func parseDuration(text string) (*time.Time, error) {
	// فرمت: 1d, 2h, 30m یا عدد به روز
	text = strings.TrimSpace(text)
	var dur time.Duration

	if strings.HasSuffix(text, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(text, "d"))
		if err != nil {
			return nil, fmt.Errorf("invalid format")
		}
		dur = time.Duration(n) * 24 * time.Hour
	} else if strings.HasSuffix(text, "h") {
		n, err := strconv.Atoi(strings.TrimSuffix(text, "h"))
		if err != nil {
			return nil, fmt.Errorf("invalid format")
		}
		dur = time.Duration(n) * time.Hour
	} else {
		n, err := strconv.Atoi(text)
		if err != nil {
			return nil, fmt.Errorf("invalid format")
		}
		dur = time.Duration(n) * 24 * time.Hour
	}

	t := time.Now().Add(dur)
	return &t, nil
}

// ── code token generator ───────────────────────────────────

func genCode() string {
	// ۶ کاراکتر عددی قابل تشخیص
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}

func genAlphaCode() string {
	// ۸ کاراکتر ترکیبی برای کدهای غیرعددی
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 8)
	seed := time.Now().UnixNano()
	for i := range code {
		code[i] = chars[seed%int64(len(chars))]
		seed = seed*6364136223846793005 + 1442695040888963407
	}
	return string(code)
}

// fileTypeIcon آیکون نوع فایل.
func fileTypeIcon(t string) string {
	m := map[string]string{
		"document":   "📄",
		"video":      "🎬",
		"audio":      "🎵",
		"photo":      "🖼",
		"voice":      "🎤",
		"animation":  "🎞",
		"video_note": "🎥",
	}
	if icon, ok := m[t]; ok {
		return icon
	}
	return "📎"
}

// codeTypeLabel نام فارسی نوع کد.
func codeTypeLabel(t documents.CodeType) string {
	m := map[documents.CodeType]string{
		documents.CodeOnce:      "یک‌بار",
		documents.CodeLimited:   "محدود",
		documents.CodeUnlimited: "نامحدود",
		documents.CodeExpiry:    "زمان‌دار",
	}
	if l, ok := m[t]; ok {
		return l
	}
	return string(t)
}

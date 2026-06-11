package tgbot

import (
	"crypto/rand"
	"encoding/hex"

	tele "gopkg.in/telebot.v4"
	"github.com/mrjvadi/creatorbot/shared-core/documents"
)

func generateCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func extractFile(c tele.Context) (fileID, fileType string) {
	m := c.Message()
	switch {
	case m.Document != nil:   return m.Document.FileID, "document"
	case m.Video != nil:      return m.Video.FileID, "video"
	case m.Audio != nil:      return m.Audio.FileID, "audio"
	case m.Voice != nil:      return m.Voice.FileID, "voice"
	case m.Animation != nil:  return m.Animation.FileID, "animation"
	case m.VideoNote != nil:  return m.VideoNote.FileID, "video_note"
	case m.Sticker != nil:    return m.Sticker.FileID, "sticker"
	case m.Photo != nil:
		p := m.Photo
		return p[len(p)-1].FileID, "photo"
	}
	return "", ""
}

func sendFile(c tele.Context, f documents.File) error {
	file := tele.FromFileID(f.TelegramFileID)
	switch f.FileType {
	case "video":     return c.Send(&tele.Video{File: file, Caption: f.Caption})
	case "audio":     return c.Send(&tele.Audio{File: file, Caption: f.Caption})
	case "photo":     return c.Send(&tele.Photo{File: file, Caption: f.Caption})
	case "voice":     return c.Send(&tele.Voice{File: file})
	case "animation": return c.Send(&tele.Animation{File: file, Caption: f.Caption})
	case "video_note":return c.Send(&tele.VideoNote{File: file})
	case "sticker":   return c.Send(&tele.Sticker{File: file})
	default:          return c.Send(&tele.Document{File: file, Caption: f.Caption})
	}
}

package userbot

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/source-service/internal/botmanager"
	"github.com/mrjvadi/creatorbot/source-service/internal/models"
)

// RunBotCommandPayload is "go press command in bot_username, take whatever
// file it replies with, and hand it off". Give this task an Envelope.ID if
// you want to correlate botmanager's eventual report with the instruction
// that triggered it.
type RunBotCommandPayload struct {
	BotUsername string `json:"bot_username"`
	Command     string `json:"command"`
}

type RunBotCommandResult struct {
	ArchiveFileID string `json:"archive_file_id"`
	FileName      string `json:"file_name"`
	MimeType      string `json:"mime_type"`
	FileSize      int64  `json:"file_size"`
}

// handleRunBotCommand sends Command to BotUsername, downloads whatever file
// it replies with, registers it in our own archive (retrievable later via
// source.files.get), and reports the result to botmanager tagged with this
// task's correlation id (see internal/botmanager.Report) so botmanager can
// match it back to whatever asked for it.
//
// ⚠️ Note on "file_id": a Telegram Bot-API file_id is minted by the Bot API
// using the bot's own token — this worker talks MTProto as a user account,
// not the Bot API, so it cannot produce a real file_id itself. Instead it
// registers the raw file here and reports the resulting archive_file_id;
// botmanager (which does hold bot tokens) is expected to fetch it via
// source.files.get and upload it through the Bot API itself to mint/cache
// the real file_id (source.files.cache already exists for that).
func (u *Userbot) handleRunBotCommand(ctx context.Context, id string, raw json.RawMessage) (any, error) {
	var p RunBotCommandPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.BotUsername == "" || p.Command == "" {
		return nil, errors.New("bot_username and command are required")
	}

	data, fileName, mimeType, replyMessageID, err := u.tg.FetchFromBot(ctx, p.BotUsername, p.Command)
	if err != nil {
		return nil, err
	}

	f := &models.ArchiveFile{
		MessageID: replyMessageID,
		FileType:  "document",
		FileName:  fileName,
		MimeType:  mimeType,
		FileSize:  int64(len(data)),
	}
	if err := u.store.CreateArchiveFile(ctx, f); err != nil {
		return nil, err
	}

	result := RunBotCommandResult{
		ArchiveFileID: f.ID.String(),
		FileName:      fileName,
		MimeType:      mimeType,
		FileSize:      f.FileSize,
	}

	tags := map[string]any{
		"action":          "bot_file_ready",
		"bot_username":    p.BotUsername,
		"archive_file_id": result.ArchiveFileID,
		"file_name":       fileName,
		"mime_type":       mimeType,
		"file_size":       result.FileSize,
	}
	if err := botmanager.Report(ctx, u.nc, u.bmCfg, id, tags); err != nil {
		u.log.Error("report to botmanager", ports.F("id", id), ports.F("err", err))
	}

	return result, nil
}

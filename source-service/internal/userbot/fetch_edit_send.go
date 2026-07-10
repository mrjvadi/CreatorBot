package userbot

import (
	"context"
	"encoding/json"
	"errors"
)

type FetchEditSendPayload struct {
	BotUsername  string `json:"bot_username"`  // bot to message, e.g. "@some_file_bot"
	Command      string `json:"command"`       // message/command to send the bot to trigger the file
	Caption      string `json:"caption"`       // new caption for the re-sent file
	DestUsername string `json:"dest_username"` // where to send the (re-captioned) file
}

func (u *Userbot) handleFetchEditSend(ctx context.Context, _ string, raw json.RawMessage) (any, error) {
	var p FetchEditSendPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.BotUsername == "" || p.Command == "" || p.DestUsername == "" {
		return nil, errors.New("bot_username, command and dest_username are required")
	}

	if err := u.tg.FetchEditSend(ctx, p.BotUsername, p.Command, p.Caption, p.DestUsername); err != nil {
		return nil, err
	}
	return map[string]bool{"sent": true}, nil
}

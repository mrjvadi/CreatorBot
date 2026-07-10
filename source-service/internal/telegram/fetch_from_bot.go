package telegram

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
)

// FetchFromBot sends `command` to botUsername and returns the file from its
// reply (document or photo) as raw bytes plus basic metadata, along with
// the bot's reply message ID. This is the shared "get a file out of a bot"
// primitive: FetchEditSend re-sends the result elsewhere; run_bot_command
// (internal/userbot) registers it and reports it to botmanager instead.
func (c *Client) FetchFromBot(ctx context.Context, botUsername, command string) (data []byte, fileName, mimeType string, replyMessageID int, err error) {
	botPeer, err := c.resolvePeer(ctx, botUsername)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("resolve bot: %w", err)
	}
	botUserPeer, ok := botPeer.(*tg.InputPeerUser)
	if !ok {
		return nil, "", "", 0, fmt.Errorf("%q is not a bot/user", botUsername)
	}

	if _, err := c.api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     botPeer,
		Message:  command,
		RandomID: randID(),
	}); err != nil {
		return nil, "", "", 0, fmt.Errorf("send command to bot: %w", err)
	}

	msg, err := c.waiter.await(ctx, botUserPeer.UserID)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("waiting for bot reply: %w", err)
	}

	loc, name, mime, err := mediaLocation(msg.Media)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("bot reply has no usable file: %w", err)
	}

	var buf bytes.Buffer
	if _, err := downloader.NewDownloader().Download(c.api, loc).Stream(ctx, &buf); err != nil {
		return nil, "", "", 0, fmt.Errorf("download file: %w", err)
	}

	return buf.Bytes(), name, mime, msg.ID, nil
}

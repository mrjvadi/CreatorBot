package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

// FetchEditSend gets a file from a bot (FetchFromBot) and re-uploads it with
// newCaption to destUsername. This is the "get the file from a bot, edit
// it, and send it" task.
//
// "Edit" here means replacing the caption. If real media editing
// (re-encoding, watermarking, cropping, ...) is needed later, that step
// goes right after FetchFromBot and before the upload call below — this
// function is the seam for it.
func (c *Client) FetchEditSend(ctx context.Context, botUsername, command, newCaption, destUsername string) error {
	data, fileName, mimeType, _, err := c.FetchFromBot(ctx, botUsername, command)
	if err != nil {
		return err
	}

	destPeer, err := c.resolvePeer(ctx, destUsername)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}

	up, err := uploader.NewUploader(c.api).FromBytes(ctx, fileName, data)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	if _, err := c.api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer:     destPeer,
		Media:    &tg.InputMediaUploadedDocument{File: up, MimeType: mimeType},
		Message:  newCaption,
		RandomID: randID(),
	}); err != nil {
		return fmt.Errorf("send file to destination: %w", err)
	}

	return nil
}

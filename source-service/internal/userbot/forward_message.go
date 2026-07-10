package userbot

import (
	"context"
	"encoding/json"
	"errors"
)

type ForwardMessagePayload struct {
	SourceChannel string `json:"source_channel"`
	DestChannel   string `json:"dest_channel"`
	MessageID     int    `json:"message_id"`
}

type ForwardMessageResult struct {
	DeliveryMessageID int `json:"delivery_message_id"`
}

func (u *Userbot) handleForwardMessage(ctx context.Context, _ string, raw json.RawMessage) (any, error) {
	var p ForwardMessagePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.SourceChannel == "" || p.DestChannel == "" || p.MessageID == 0 {
		return nil, errors.New("source_channel, dest_channel and message_id are required")
	}

	deliveryID, err := u.tg.ForwardMessage(ctx, p.SourceChannel, p.DestChannel, p.MessageID)
	if err != nil {
		return nil, err
	}
	return ForwardMessageResult{DeliveryMessageID: deliveryID}, nil
}

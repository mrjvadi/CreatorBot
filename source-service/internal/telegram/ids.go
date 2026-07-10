package telegram

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gotd/td/tg"
)

var (
	errNoDocument  = errors.New("document not available")
	errNoPhoto     = errors.New("photo not available")
	errNoPhotoSize = errors.New("no usable photo size")
)

func errUnsupportedMedia(media tg.MessageMediaClass) error {
	return fmt.Errorf("unsupported media type %T", media)
}

// randID generates a random int64, as MTProto request IDs require.
func randID() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int64(binary.BigEndian.Uint64(b[:]))
}

// newMessageID pulls the newly created message's ID out of an Updates
// response, if present.
func newMessageID(u tg.UpdatesClass) int {
	updates, ok := u.(*tg.Updates)
	if !ok {
		return 0
	}
	for _, up := range updates.Updates {
		if m, ok := up.(*tg.UpdateNewMessage); ok {
			if msg, ok := m.Message.(*tg.Message); ok {
				return msg.ID
			}
		}
	}
	return 0
}

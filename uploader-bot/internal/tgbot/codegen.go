package tgbot

import (
	"crypto/rand"
	"encoding/hex"
)

func generateCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

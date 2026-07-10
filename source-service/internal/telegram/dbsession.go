package telegram

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/mrjvadi/creatorbot/source-service/internal/store"
)

// DBSessionStorage implements gotd/td's session.Storage on top of Postgres
// (via internal/store) instead of a file, so a lost Docker volume doesn't
// force a fresh Telegram login: the session survives in the database. The
// session is AES-256-GCM encrypted at rest; the key comes from botmanager's
// registration reply (internal/worker.TelegramCreds.SessionKey) and never
// touches disk unencrypted or in plaintext logs.
type DBSessionStorage struct {
	store *store.Store
	phone string // normalized (digits-only)
	key   []byte // 32 bytes, AES-256
}

// NewDBSessionStorage builds a DBSessionStorage for one account. base64Key
// must decode to exactly 32 bytes (AES-256) — generate one with
// `go run ./cmd/login --gen-key`.
func NewDBSessionStorage(st *store.Store, phone, base64Key string) (*DBSessionStorage, error) {
	key, err := decodeSessionKey(base64Key)
	if err != nil {
		return nil, fmt.Errorf("session encryption key: %w", err)
	}
	return &DBSessionStorage{store: st, phone: NormalizePhone(phone), key: key}, nil
}

func decodeSessionKey(b64 string) ([]byte, error) {
	if b64 == "" {
		return nil, fmt.Errorf("key is empty")
	}
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key must decode to 32 bytes (AES-256), got %d", len(key))
	}
	return key, nil
}

// GenerateSessionKey returns a fresh, random base64-encoded AES-256 key —
// used by `cmd/login --gen-key` and, longer-term, whatever in botmanager
// mints keys for new licenses.
func GenerateSessionKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// LoadSession implements gotd/td's session.Storage. Returning (nil, nil)
// tells gotd/td there's no session yet — that's the expected first run.
func (s *DBSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	encrypted, err := s.store.GetTelegramSession(ctx, s.phone)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return s.decrypt(encrypted)
}

// StoreSession implements gotd/td's session.Storage.
func (s *DBSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	encrypted, err := s.encrypt(data)
	if err != nil {
		return err
	}
	return s.store.UpsertTelegramSession(ctx, s.phone, encrypted)
}

func (s *DBSessionStorage) encrypt(plaintext []byte) ([]byte, error) {
	gcm, err := s.gcm()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	// Nonce is prepended to the ciphertext so LoadSession only needs to
	// store/read one blob.
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (s *DBSessionStorage) decrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	gcm, err := s.gcm()
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("encrypted session data is corrupt (too short)")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *DBSessionStorage) gcm() (cipher.AEAD, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

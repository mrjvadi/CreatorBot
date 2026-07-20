// This file holds the archive file registry: register/get/cache, kept over
// from before the worker/task pivot (see README "ثبت فایل آرشیو").
package bus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/source-service/internal/models"
)

// Request-reply subjects for the archive file registry.
const (
	SubjectFilesRegister = "source.files.register"
	SubjectFilesGet      = "source.files.get"
	SubjectFilesCache    = "source.files.cache"
)

// EventFileRegistered is published after a file is registered.
const EventFileRegistered = "source.files.registered"

type RequestAuth struct {
	ServiceID  string `json:"service_id"`
	ServiceKey string `json:"service_key"`
	TenantID   string `json:"tenant_id"`
	IssuedAt   int64  `json:"issued_at"`
	Nonce      string `json:"nonce"`
}

type RegisterRequest struct {
	RequestAuth
	MessageID int    `json:"message_id"`
	FileType  string `json:"file_type"`
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	FileSize  int64  `json:"file_size"`
	Caption   string `json:"caption"`
}

type GetRequest struct {
	RequestAuth
	UUID uuid.UUID `json:"uuid"`
	// BotTokenHash is optional. When set, the reply also includes the
	// file_id this bot previously cached for the file, if any.
	BotTokenHash string `json:"bot_token_hash,omitempty"`
}

type CacheRequest struct {
	RequestAuth
	UUID         uuid.UUID `json:"uuid"`
	BotTokenHash string    `json:"bot_token_hash"`
	FileID       string    `json:"file_id"`
}

// FileEnvelope is the reply shape for all three file-registry subjects.
type FileEnvelope struct {
	OK           bool                `json:"ok"`
	Error        string              `json:"error,omitempty"`
	File         *models.ArchiveFile `json:"file,omitempty"`
	CachedFileID string              `json:"cached_file_id,omitempty"`
}

func (b *Bus) handleRegister(ctx context.Context, data []byte) (any, error) {
	var req RegisterRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return errEnvelope(err), nil
	}
	if err := b.authorize(ctx, req.ServiceID, req.ServiceKey, req.TenantID, req.IssuedAt, req.Nonce); err != nil {
		return errEnvelope(err), nil
	}

	f := &models.ArchiveFile{
		TenantID:  req.TenantID,
		MessageID: req.MessageID,
		FileType:  req.FileType,
		FileName:  req.FileName,
		MimeType:  req.MimeType,
		FileSize:  req.FileSize,
		Caption:   req.Caption,
	}
	if err := b.store.CreateArchiveFile(ctx, f); err != nil {
		return errEnvelope(err), nil
	}

	if err := b.nc.PublishCore(EventFileRegistered, f); err != nil {
		b.log.Error("publish registered event", ports.F("err", err))
	}

	return FileEnvelope{OK: true, File: f}, nil
}

func (b *Bus) handleGet(ctx context.Context, data []byte) (any, error) {
	var req GetRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return errEnvelope(err), nil
	}
	if err := b.authorize(ctx, req.ServiceID, req.ServiceKey, req.TenantID, req.IssuedAt, req.Nonce); err != nil {
		return errEnvelope(err), nil
	}

	f, err := b.store.GetArchiveFile(ctx, req.TenantID, req.UUID)
	if err != nil {
		return errEnvelope(err), nil
	}

	resp := FileEnvelope{OK: true, File: f}

	if req.BotTokenHash != "" {
		key := cacheKeyFor(req.UUID, req.BotTokenHash)
		if fileID, err := b.cache.Get(ctx, key); err == nil && fileID != "" {
			resp.CachedFileID = fileID
		} else if c, err := b.store.GetBotFileCache(ctx, req.TenantID, req.UUID, req.BotTokenHash); err == nil {
			resp.CachedFileID = c.FileID
			if setErr := b.cache.Set(ctx, key, c.FileID, 24*time.Hour); setErr != nil {
				b.log.Error("cache set", ports.F("err", setErr))
			}
		}
	}

	return resp, nil
}

func (b *Bus) handleCache(ctx context.Context, data []byte) (any, error) {
	var req CacheRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return errEnvelope(err), nil
	}
	if err := b.authorize(ctx, req.ServiceID, req.ServiceKey, req.TenantID, req.IssuedAt, req.Nonce); err != nil {
		return errEnvelope(err), nil
	}

	if _, err := b.store.GetArchiveFile(ctx, req.TenantID, req.UUID); err != nil {
		return errEnvelope(err), nil
	}
	c := &models.BotFileCache{
		TenantID:      req.TenantID,
		ArchiveFileID: req.UUID,
		BotTokenHash:  req.BotTokenHash,
		FileID:        req.FileID,
		CachedAt:      time.Now(),
	}
	if err := b.store.UpsertBotFileCache(ctx, c); err != nil {
		return errEnvelope(err), nil
	}

	if err := b.cache.Set(ctx, cacheKeyFor(req.UUID, req.BotTokenHash), req.FileID, 24*time.Hour); err != nil {
		b.log.Error("cache set", ports.F("err", err))
	}

	return FileEnvelope{OK: true}, nil
}

func cacheKeyFor(id uuid.UUID, botTokenHash string) string {
	return "source:filecache:" + id.String() + ":" + botTokenHash
}

// errEnvelope always produces a valid FileEnvelope (ok:false) rather than a
// Go error, so the shared NATS client's Respond doesn't override it with
// its own generic {"error":"..."} shape — callers always get a structured
// reason inside the normal envelope.
func errEnvelope(err error) FileEnvelope {
	return FileEnvelope{OK: false, Error: err.Error()}
}

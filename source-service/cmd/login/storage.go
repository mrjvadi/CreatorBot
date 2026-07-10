package main

import (
	"fmt"
	"path/filepath"

	"github.com/gotd/td/session"

	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"

	"github.com/mrjvadi/creatorbot/source-service/internal/models"
	"github.com/mrjvadi/creatorbot/source-service/internal/store"
	"github.com/mrjvadi/creatorbot/source-service/internal/telegram"
)

// buildSessionStorage picks between the local-file fallback (--file) and
// the encrypted Postgres storage real workers use (default), and returns a
// short human-readable description of which one was picked.
func buildSessionStorage(f flags) (session.Storage, string, error) {
	if f.useFile {
		sessionFile := filepath.Join(f.sessionsDir, telegram.SessionFileName(f.phone))
		return &session.FileStorage{Path: sessionFile}, "Using local file storage: " + sessionFile, nil
	}

	if f.postgresDSN == "" || f.sessionKey == "" {
		return nil, "", fmt.Errorf("--postgres-dsn and --session-key (or $POSTGRES_DSN / $SESSION_ENCRYPTION_KEY) are required unless --file is set")
	}

	db, err := postgres.New(postgres.Config{DSN: f.postgresDSN})
	if err != nil {
		return nil, "", fmt.Errorf("connect to postgres: %w", err)
	}
	db.Migrate(&models.TelegramSession{})
	st := store.New(db)

	dbStorage, err := telegram.NewDBSessionStorage(st, f.phone, f.sessionKey)
	if err != nil {
		return nil, "", fmt.Errorf("session storage: %w", err)
	}
	return dbStorage, "Using encrypted database storage (same as production workers)", nil
}

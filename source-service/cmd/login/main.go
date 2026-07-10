// cmd/login is a one-off, interactive tool for logging a Telegram account
// in by hand: give it an app id/hash and a phone number, type in the code
// Telegram sends you, and it saves the MTProto session. It's meant for
// testing and emergency recovery, not the normal path — normally a worker
// (cmd/service) logs in on its own using the code botmanager relays over
// NATS.
//
// By default it saves the session the same encrypted way a real worker
// would (Postgres, AES-256-GCM, see storage.go) so a recovery login
// actually restores what workers will read — pass --file to use a plain
// local session file instead, for quick offline testing without a
// database.
//
// Usage:
//
//	# generate a session encryption key once (or get one from botmanager)
//	go run ./cmd/login --gen-key
//
//	# log in, storing the (encrypted) session in Postgres like production:
//	go run ./cmd/login --app-id 12345 --app-hash xxxxx --phone +989120000000 \
//	  --postgres-dsn "$POSTGRES_DSN" --session-key "$SESSION_ENCRYPTION_KEY"
//
//	# or a plain local file, for quick offline testing:
//	go run ./cmd/login --app-id 12345 --app-hash xxxxx --phone +989120000000 --file
//
// via docker compose (see deploy/docker-compose.yml, "login" service):
//
//	docker compose run --rm login --app-id 12345 --app-hash xxxx --phone +989120000000
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mrjvadi/creatorbot/shared/pkg/logger"

	"github.com/mrjvadi/creatorbot/source-service/internal/telegram"
)

func main() {
	flags := parseFlags()

	if flags.genKey {
		key, err := telegram.GenerateSessionKey()
		if err != nil {
			fmt.Fprintln(os.Stderr, "generate key:", err)
			os.Exit(1)
		}
		fmt.Println(key)
		return
	}

	if flags.appID == 0 || flags.appHash == "" {
		fmt.Fprintln(os.Stderr, "--app-id and --app-hash are required (from my.telegram.org/apps), or set TG_APP_ID/TG_APP_HASH")
		os.Exit(1)
	}
	if flags.phone == "" {
		flags.phone = prompt("Phone number (e.g. +989120000000): ")
	}
	if flags.phone == "" {
		fmt.Fprintln(os.Stderr, "a phone number is required")
		os.Exit(1)
	}

	log := logger.MustNew(false)

	sessionStorage, description, err := buildSessionStorage(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(description)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client := telegram.New(telegram.Config{
		AppID:          flags.appID,
		AppHash:        flags.appHash,
		Phone:          flags.phone,
		SessionStorage: sessionStorage,
	}, telegram.StdinCodeSource{}, log)

	fmt.Println("Connecting to Telegram...")
	if err := client.Login(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "login failed:", err)
		os.Exit(1)
	}
	fmt.Println("Logged in and session saved for", flags.phone)
	if !flags.useFile {
		fmt.Println("Any worker registered with this same phone number (and the same --session-key) will reuse this session automatically.")
	}
}

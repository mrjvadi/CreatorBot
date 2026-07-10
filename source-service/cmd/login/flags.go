package main

import (
	"flag"
	"os"
)

type flags struct {
	appID       int
	appHash     string
	phone       string
	genKey      bool
	useFile     bool
	sessionsDir string
	postgresDSN string
	sessionKey  string
}

func parseFlags() flags {
	f := flags{}
	flag.IntVar(&f.appID, "app-id", 0, "Telegram app id (my.telegram.org/apps)")
	flag.StringVar(&f.appHash, "app-hash", "", "Telegram app hash (my.telegram.org/apps)")
	flag.StringVar(&f.phone, "phone", "", "phone number, e.g. +989120000000")
	flag.BoolVar(&f.genKey, "gen-key", false, "print a fresh SESSION_ENCRYPTION_KEY and exit")
	flag.BoolVar(&f.useFile, "file", false, "save the session to a local file instead of Postgres (offline testing only)")
	flag.StringVar(&f.sessionsDir, "sessions-dir", envOr("SESSIONS_DIR", "/app/sessions"), "directory for --file mode")
	flag.StringVar(&f.postgresDSN, "postgres-dsn", os.Getenv("POSTGRES_DSN"), "Postgres DSN (defaults to $POSTGRES_DSN)")
	flag.StringVar(&f.sessionKey, "session-key", os.Getenv("SESSION_ENCRYPTION_KEY"), "base64 AES-256 key (defaults to $SESSION_ENCRYPTION_KEY)")
	flag.Parse()

	if f.appID == 0 {
		f.appID = envInt("TG_APP_ID")
	}
	if f.appHash == "" {
		f.appHash = os.Getenv("TG_APP_HASH")
	}
	return f
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string) int {
	v := os.Getenv(key)
	n := 0
	for i := 0; i < len(v); i++ {
		if v[i] < '0' || v[i] > '9' {
			return 0
		}
		n = n*10 + int(v[i]-'0')
	}
	return n
}

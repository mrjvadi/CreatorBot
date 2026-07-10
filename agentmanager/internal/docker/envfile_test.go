package docker

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMergeEnv_OverlayWins(t *testing.T) {
	base := map[string]string{
		"NATS_URL":  "nats://nats:4222",
		"MONGO_URI": "mongodb://mongodb:27017",
		"REDIS_DB":  "0",
	}
	overlay := map[string]string{
		"BOT_TOKEN": "123:abc",
		"REDIS_DB":  "5", // باید روی base برنده شود
	}
	got := mergeEnv(base, overlay)
	want := []string{
		"BOT_TOKEN=123:abc",
		"MONGO_URI=mongodb://mongodb:27017",
		"NATS_URL=nats://nats:4222",
		"REDIS_DB=5",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeEnv:\n got=%v\nwant=%v", got, want)
	}
}

func TestMergeEnv_NilBase(t *testing.T) {
	got := mergeEnv(nil, map[string]string{"A": "1"})
	if !reflect.DeepEqual(got, []string{"A=1"}) {
		t.Fatalf("got %v", got)
	}
}

func TestMergeEnv_Empty(t *testing.T) {
	if got := mergeEnv(nil, nil); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestParseEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "botenv.env")
	content := "# کامنت\nNATS_URL=nats://nats:4222\n\nMONGO_URI=mongodb://u:p@mongodb:27017/db?authSource=admin\nEMPTY_VAL=\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"NATS_URL":  "nats://nats:4222",
		"MONGO_URI": "mongodb://u:p@mongodb:27017/db?authSource=admin",
		"EMPTY_VAL": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseEnvFile:\n got=%v\nwant=%v", got, want)
	}
}

func TestParseEnvFile_InvalidLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.env")
	if err := os.WriteFile(path, []byte("NOT A KV LINE\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseEnvFile(path); err == nil {
		t.Fatal("expected error for invalid line")
	}
}

func TestParseEnvFile_Missing(t *testing.T) {
	if _, err := ParseEnvFile(filepath.Join(t.TempDir(), "nope.env")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

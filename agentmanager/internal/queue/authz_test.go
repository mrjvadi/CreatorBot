package queue

import (
	"testing"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

const testSecret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// signed یک دستور معتبرِ امضاشده با مقادیر دلخواه می‌سازد.
func signed(serviceID string, issuedAt int64, nonce string) protocol.DeployCommand {
	return protocol.DeployCommand{
		Type:          protocol.MsgDeploy,
		ContainerName: "uploader_1",
		ServiceID:     serviceID,
		ServiceKey:    auth.ComputeServiceKey(testSecret, serviceID),
		IssuedAt:      issuedAt,
		Nonce:         nonce,
	}
}

func TestVerifier_ValidCommandPasses(t *testing.T) {
	v := NewVerifier(testSecret)
	if err := v.Check(signed("botmanager", time.Now().Unix(), "n-valid")); err != nil {
		t.Fatalf("expected valid command to pass, got: %v", err)
	}
}

func TestVerifier_BadKeyRejected(t *testing.T) {
	v := NewVerifier(testSecret)
	cmd := signed("botmanager", time.Now().Unix(), "n-badkey")
	cmd.ServiceKey = "deadbeef" // tampered
	if err := v.Check(cmd); err == nil {
		t.Fatal("expected rejection for bad service key")
	}
}

func TestVerifier_EmptySecretRejectsEverything(t *testing.T) {
	v := NewVerifier("")
	if err := v.Check(signed("botmanager", time.Now().Unix(), "n-empty")); err == nil {
		t.Fatal("expected fail-closed rejection when secret is empty")
	}
}

func TestVerifier_StaleRejected(t *testing.T) {
	v := NewVerifier(testSecret)
	old := time.Now().Add(-6 * time.Minute).Unix()
	if err := v.Check(signed("botmanager", old, "n-stale")); err == nil {
		t.Fatal("expected rejection for stale issued_at (>5m)")
	}
}

func TestVerifier_FutureRejected(t *testing.T) {
	v := NewVerifier(testSecret)
	future := time.Now().Add(60 * time.Second).Unix()
	if err := v.Check(signed("botmanager", future, "n-future")); err == nil {
		t.Fatal("expected rejection for issued_at too far in the future")
	}
}

func TestVerifier_ReplayRejected(t *testing.T) {
	v := NewVerifier(testSecret)
	cmd := signed("botmanager", time.Now().Unix(), "n-replay")
	if err := v.Check(cmd); err != nil {
		t.Fatalf("first use should pass, got: %v", err)
	}
	if err := v.Check(cmd); err == nil {
		t.Fatal("expected rejection on replay of the same nonce")
	}
}

func TestVerifier_EmptyNonceRejected(t *testing.T) {
	v := NewVerifier(testSecret)
	if err := v.Check(signed("botmanager", time.Now().Unix(), "")); err == nil {
		t.Fatal("expected rejection for empty nonce")
	}
}

// نمونه‌ی fresh nonce از دو serviceID متفاوت نباید با هم تداخل کند.
func TestVerifier_NonceScopedByService(t *testing.T) {
	v := NewVerifier(testSecret)
	now := time.Now().Unix()
	if err := v.Check(signed("botmanager", now, "shared")); err != nil {
		t.Fatalf("botmanager should pass: %v", err)
	}
	if err := v.Check(signed("apimanager", now, "shared")); err != nil {
		t.Fatalf("apimanager with same nonce string but different service should pass: %v", err)
	}
}

package responder

import (
	"testing"

	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

func TestAuthorizeServiceAllowsBothControlPlanes(t *testing.T) {
	const secret = "test-service-hmac-secret"
	r := &Responder{serviceHMACSecret: secret}
	for _, serviceID := range []string{"agentmanager", "botmanager", "apimanager"} {
		t.Run(serviceID, func(t *testing.T) {
			key := auth.ComputeServiceKey(secret, serviceID)
			if !r.authorizeService(serviceID, key) {
				t.Fatalf("expected %s to be authorized", serviceID)
			}
		})
	}
}

func TestAuthorizeServiceRejectsUnknownOrWrongKey(t *testing.T) {
	r := &Responder{serviceHMACSecret: "secret"}
	if r.authorizeService("apimanager", "wrong") {
		t.Fatal("wrong key must be rejected")
	}
	if r.authorizeService("unknown", auth.ComputeServiceKey("secret", "unknown")) {
		t.Fatal("unknown service must be rejected")
	}
}

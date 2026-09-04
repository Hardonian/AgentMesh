package crypto_test

import (
	"errors"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/crypto"
)

func TestSigningAndVerification(t *testing.T) {
	kp, err := crypto.GenerateKeyPair("key_v1")
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	keyRing := crypto.NewKeyRing()
	keyRing.RegisterKey("key_v1", kp.PublicKey)

	payload := map[string]string{
		"agent":  "finance-agent",
		"status": "healthy",
	}

	bundle, err := crypto.SignPayload(kp, "v1.0.0", 1*time.Hour, payload)
	if err != nil {
		t.Fatalf("failed to sign payload: %v", err)
	}

	// 1. Valid bundle verification
	if err := keyRing.Verify(bundle); err != nil {
		t.Errorf("verification failed for valid bundle: %v", err)
	}

	// 2. Tampered payload detection
	tamperedBundle := *bundle
	tamperedBundle.Payload = `{"agent":"evil-agent"}`
	if err := keyRing.Verify(&tamperedBundle); err == nil {
		t.Error("expected verification failure on tampered bundle, got nil")
	}

	// 3. Expired bundle
	expiredBundle, _ := crypto.SignPayload(kp, "v1.0.0", -1*time.Minute, payload)
	if err := keyRing.Verify(expiredBundle); err == nil {
		t.Error("expected verification failure on expired bundle, got nil")
	}

	// 4. Key rotation
	kp2, _ := crypto.GenerateKeyPair("key_v2")
	keyRing.RegisterKey("key_v2", kp2.PublicKey)

	bundle2, _ := crypto.SignPayload(kp2, "v2.0.0", 1*time.Hour, payload)
	if err := keyRing.Verify(bundle2); err != nil {
		t.Errorf("verification failed for rotated key bundle: %v", err)
	}

	// 5. Future-issued bundle rejection
	futureBundle := *bundle
	futureBundle.IssuedAt = time.Now().UTC().Add(10 * time.Minute)
	// re-sign with future timestamp
	futureSigned, _ := crypto.SignPayload(kp, "v1.0.0", 1*time.Hour, payload)
	futureSigned.IssuedAt = time.Now().UTC().Add(10 * time.Minute)
	if err := keyRing.Verify(futureSigned); !errors.Is(err, crypto.ErrBundleFutureIssued) {
		t.Errorf("expected ErrBundleFutureIssued for future-dated bundle, got: %v", err)
	}
}

package crypto_test

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/crypto"
)

func TestCloudKMSSigner_SignAndVerify(t *testing.T) {
	kmsName := "projects/acme-gcp/locations/us-central1/keyRings/agentmesh-ring/cryptoKeys/proxy-signer/cryptoKeyVersions/1"
	signer, err := crypto.NewCloudKMSSigner(kmsName, crypto.AlgorithmEd25519, true)
	if err != nil {
		t.Fatalf("NewCloudKMSSigner failed: %v", err)
	}

	payload := map[string]any{
		"agents": []string{"finance-agent", "research-agent"},
		"routes": 4,
	}

	bundle, err := signer.SignConfigBundle(context.Background(), "v2.1", 10*time.Minute, payload)
	if err != nil {
		t.Fatalf("SignConfigBundle failed: %v", err)
	}

	if bundle.KeyID != kmsName {
		t.Fatalf("expected key ID to match KMS key version name, got: %s", bundle.KeyID)
	}
	if bundle.Signature == "" {
		t.Fatal("expected non-empty signature")
	}

	// Verify cryptographic signature against public key
	sigBytes, err := hex.DecodeString(bundle.Signature)
	if err != nil {
		t.Fatalf("invalid hex signature: %v", err)
	}

	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%d:%s",
		bundle.Version, bundle.KeyID, bundle.IssuedAt.Unix(), bundle.ExpiresAt.Unix(), bundle.Payload)))

	pubKey := ed25519.PublicKey(signer.GetPublicKey())
	if !ed25519.Verify(pubKey, digest[:], sigBytes) {
		t.Fatal("KMS bundle signature verification failed")
	}
}

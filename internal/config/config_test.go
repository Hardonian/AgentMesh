package config_test

import (
	"errors"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/config"
	"github.com/agentmesh/agentmesh/internal/crypto"
	"github.com/agentmesh/agentmesh/internal/policy"
)

func TestProxyConfigCache_LKGAndDowngradeProtection(t *testing.T) {
	kp, err := crypto.GenerateKeyPair("signer_key")
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	keyRing := crypto.NewKeyRing()
	keyRing.RegisterKey("signer_key", kp.PublicKey)

	engine := policy.NewEngine([]*policy.Policy{})
	cache := config.NewProxyConfigCache(keyRing, engine)

	pol1 := &policy.Policy{
		ID:       "pol_v1",
		Version:  "1.0",
		TenantID: "tenant-a",
	}

	// 1. Initial valid bundle
	b1, err := crypto.SignPayload(kp, "v1", 1*time.Hour, map[string]string{"rule": "allow"})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	if err := cache.UpdateFromBundle(b1, pol1); err != nil {
		t.Fatalf("initial bundle update failed: %v", err)
	}
	if cache.CachedPolicy() != pol1 {
		t.Errorf("expected cached policy pol1, got %v", cache.CachedPolicy())
	}

	// 2. Tampered bundle: cache must retain Last Known Good (pol1)
	tampered := *b1
	tampered.Payload = "tampered"
	err = cache.UpdateFromBundle(&tampered, nil)
	if err == nil {
		t.Fatal("expected signature verification error on tampered bundle")
	}
	if cache.CachedPolicy() != pol1 {
		t.Errorf("cache failed to retain LKG after tampered update: got %v", cache.CachedPolicy())
	}

	// 3. Newer bundle (issued after b1)
	time.Sleep(10 * time.Millisecond)
	pol2 := &policy.Policy{ID: "pol_v2", Version: "2.0", TenantID: "tenant-a"}
	b2, _ := crypto.SignPayload(kp, "v2", 1*time.Hour, map[string]string{"rule": "restrict"})
	if err := cache.UpdateFromBundle(b2, pol2); err != nil {
		t.Fatalf("newer bundle update failed: %v", err)
	}
	if cache.CachedPolicy() != pol2 {
		t.Errorf("expected cached policy pol2, got %v", cache.CachedPolicy())
	}

	// 4. Downgrade attack: replaying b1 (older timestamp) must be rejected
	err = cache.UpdateFromBundle(b1, pol1)
	if !errors.Is(err, config.ErrConfigDowngrade) {
		t.Errorf("expected ErrConfigDowngrade on replaying older bundle b1, got: %v", err)
	}
	if cache.CachedPolicy() != pol2 {
		t.Errorf("cache allowed downgrade! active policy is %v", cache.CachedPolicy())
	}

	// 5. Expiration: let bundle expire -> CachedPolicy fails closed (returns nil)
	expBundle, _ := crypto.SignPayload(kp, "v3", 5*time.Millisecond, map[string]string{"rule": "temp"})
	time.Sleep(10 * time.Millisecond) // Ensure issuedAt is after b2
	expPol := &policy.Policy{ID: "pol_exp", Version: "3.0"}
	if err := cache.UpdateFromBundle(expBundle, expPol); err != nil {
		t.Fatalf("failed to update temp bundle: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Wait for expiry
	if cache.CachedPolicy() != nil {
		t.Errorf("expected nil (fail closed) for expired cached policy, got %v", cache.CachedPolicy())
	}
}

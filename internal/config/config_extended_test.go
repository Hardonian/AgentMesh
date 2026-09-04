package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/config"
	"github.com/agentmesh/agentmesh/internal/crypto"
	"github.com/agentmesh/agentmesh/internal/policy"
)

func TestConfig_LoadFromEnv(t *testing.T) {
	// Clean environment variables for test
	envVars := []string{
		"AGENTMESH_ENV",
		"AGENTMESH_HTTP_PORT",
		"AGENTMESH_PROXY_PORT",
		"DATABASE_URL",
		"AGENTMESH_SIGNING_KEY_ID",
		"AGENTMESH_CONTROL_PLANE_URL",
		"AGENTMESH_LOG_LEVEL",
	}
	originalVals := make(map[string]string)
	for _, k := range envVars {
		originalVals[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	defer func() {
		for k, v := range originalVals {
			if v != "" {
				_ = os.Setenv(k, v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}()

	// 1. Defaults (development)
	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error on default LoadFromEnv: %v", err)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected development env, got %s", cfg.Environment)
	}
	if cfg.HTTPPort != 8080 || cfg.ProxyPort != 9090 {
		t.Errorf("unexpected ports: HTTP %d, Proxy %d", cfg.HTTPPort, cfg.ProxyPort)
	}
	if cfg.SigningKeyID != "default_local_key" {
		t.Errorf("unexpected signing key: %s", cfg.SigningKeyID)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("unexpected log level: %s", cfg.LogLevel)
	}

	// 2. Production missing DATABASE_URL should error
	_ = os.Setenv("AGENTMESH_ENV", "production")
	_, err = config.LoadFromEnv()
	if err == nil {
		t.Fatal("expected error in production mode when DATABASE_URL is missing")
	}

	// 3. Custom environment variables
	_ = os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	_ = os.Setenv("AGENTMESH_HTTP_PORT", "8181")
	_ = os.Setenv("AGENTMESH_PROXY_PORT", "9191")
	_ = os.Setenv("AGENTMESH_SIGNING_KEY_ID", "prod_key_01")
	_ = os.Setenv("AGENTMESH_CONTROL_PLANE_URL", "https://control.mesh.corp")
	_ = os.Setenv("AGENTMESH_LOG_LEVEL", "debug")

	cfgProd, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error with full env config: %v", err)
	}
	if cfgProd.Environment != "production" {
		t.Errorf("expected production, got %s", cfgProd.Environment)
	}
	if cfgProd.HTTPPort != 8181 || cfgProd.ProxyPort != 9191 {
		t.Errorf("unexpected custom ports: %d, %d", cfgProd.HTTPPort, cfgProd.ProxyPort)
	}
	if cfgProd.SigningKeyID != "prod_key_01" {
		t.Errorf("unexpected key id: %s", cfgProd.SigningKeyID)
	}
	if cfgProd.ControlPlaneURL != "https://control.mesh.corp" {
		t.Errorf("unexpected cp url: %s", cfgProd.ControlPlaneURL)
	}
	if cfgProd.LogLevel != "debug" {
		t.Errorf("unexpected log level: %s", cfgProd.LogLevel)
	}
}

func TestConfig_CacheStatus(t *testing.T) {
	keyRing := crypto.NewKeyRing()
	engine := policy.NewEngine([]*policy.Policy{})
	cache := config.NewProxyConfigCache(keyRing, engine)

	// Status on empty cache
	ver, _, hasConfig := cache.Status()
	if hasConfig || ver != "" {
		t.Errorf("expected empty cache status, got ver=%s hasConfig=%v", ver, hasConfig)
	}

	// Sign and update
	kp, _ := crypto.GenerateKeyPair("signer_key")
	keyRing.RegisterKey("signer_key", kp.PublicKey)
	bundle, _ := crypto.SignPayload(kp, "v1.2.3", 1*time.Hour, map[string]string{"foo": "bar"})
	_ = cache.UpdateFromBundle(bundle, nil)

	ver, age, hasConfig := cache.Status()
	if !hasConfig || ver != "v1.2.3" {
		t.Errorf("expected populated cache status, got ver=%s hasConfig=%v", ver, hasConfig)
	}
	if age < 0 {
		t.Errorf("unexpected negative age: %v", age)
	}
}

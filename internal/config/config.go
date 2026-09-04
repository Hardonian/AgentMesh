package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/internal/crypto"
	"github.com/agentmesh/agentmesh/internal/policy"
)

// AppConfig represents validated server configuration.
type AppConfig struct {
	Environment     string `json:"environment"` // "development", "staging", "production"
	HTTPPort        int    `json:"httpPort"`
	ProxyPort       int    `json:"proxyPort"`
	DatabaseURL     string `json:"databaseUrl"`
	SigningKeyID    string `json:"signingKeyId"`
	ControlPlaneURL string `json:"controlPlaneUrl"`
	LogLevel        string `json:"logLevel"`
}

// LoadFromEnv loads and validates environment variables.
func LoadFromEnv() (*AppConfig, error) {
	env := os.Getenv("AGENTMESH_ENV")
	if env == "" {
		env = "development"
	}

	httpPort := 8080
	if p := os.Getenv("AGENTMESH_HTTP_PORT"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			httpPort = val
		}
	}

	proxyPort := 9090
	if p := os.Getenv("AGENTMESH_PROXY_PORT"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			proxyPort = val
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" && env == "production" {
		return nil, errors.New("DATABASE_URL is required in production mode")
	}

	signingKeyID := os.Getenv("AGENTMESH_SIGNING_KEY_ID")
	if signingKeyID == "" {
		signingKeyID = "default_local_key"
	}

	cpURL := os.Getenv("AGENTMESH_CONTROL_PLANE_URL")
	if cpURL == "" {
		cpURL = fmt.Sprintf("http://127.0.0.1:%d", httpPort)
	}

	logLevel := os.Getenv("AGENTMESH_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return &AppConfig{
		Environment:     env,
		HTTPPort:        httpPort,
		ProxyPort:       proxyPort,
		DatabaseURL:     dbURL,
		SigningKeyID:    signingKeyID,
		ControlPlaneURL: cpURL,
		LogLevel:        logLevel,
	}, nil
}

// ProxyConfigCache caches signed bundles for offline survivability.
type ProxyConfigCache struct {
	mu           sync.RWMutex
	keyRing      *crypto.KeyRing
	lastValid    *crypto.SignedBundle
	cachedPolicy *policy.Policy
	engine       *policy.Engine
	lastSyncAt   time.Time
}

// NewProxyConfigCache creates a config cache.
func NewProxyConfigCache(kr *crypto.KeyRing, engine *policy.Engine) *ProxyConfigCache {
	return &ProxyConfigCache{
		keyRing: kr,
		engine:  engine,
	}
}

// UpdateFromBundle verifies signature and applies new configuration if valid.
// Invariant: If update is invalid/tampered, retain previous last-known-good.
func (c *ProxyConfigCache) UpdateFromBundle(bundle *crypto.SignedBundle, pol *policy.Policy) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cryptographic verification
	if err := c.keyRing.Verify(bundle); err != nil {
		return fmt.Errorf("failed to verify bundle signature: %w", err)
	}

	// Update active cache
	c.lastValid = bundle
	c.cachedPolicy = pol
	c.lastSyncAt = time.Now().UTC()

	if pol != nil && c.engine != nil {
		c.engine.SetPolicies([]*policy.Policy{pol})
	}

	return nil
}

// Status returns metadata about the current cached configuration.
func (c *ProxyConfigCache) Status() (version string, age time.Duration, hasConfig bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastValid == nil {
		return "", 0, false
	}
	return c.lastValid.Version, time.Since(c.lastSyncAt), true
}

// CachedPolicy returns the last known good policy.
func (c *ProxyConfigCache) CachedPolicy() *policy.Policy {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cachedPolicy
}

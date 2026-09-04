package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Predefined API Key Scopes
const (
	ScopeAgentsRead     = "agents:read"
	ScopeAgentsWrite    = "agents:write"
	ScopeAgentsInvoke   = "agents:invoke"
	ScopePoliciesRead   = "policies:read"
	ScopePoliciesWrite  = "policies:write"
	ScopeTelemetryWrite = "telemetry:write"
	ScopeApprovalsWrite = "approvals:write"
	ScopeAdmin          = "admin"
)

// Credential represents a hashed secret associated with an agent or user.
type Credential struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenantId"`
	SubjectID   string     `json:"subjectId"` // Agent ID or User ID
	KeyPrefix   string     `json:"keyPrefix"` // First 8 chars for display
	HashedKey   string     `json:"-"`         // SHA-256 hash
	Scopes      []string   `json:"scopes"`
	CreatedAt   time.Time  `json:"createdAt"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	Revoked     bool       `json:"revoked"`
	Description string     `json:"description,omitempty"`
}

// GenerateAPIKey creates a secure random API key and returns the plaintext key and the Credential record.
func GenerateAPIKey(tenantID, subjectID string, scopes []string, ttl time.Duration, description string) (string, *Credential, error) {
	rawBytes := make([]byte, 24)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	rawKey := "mesh_" + hex.EncodeToString(rawBytes)
	hash := HashKey(rawKey)
	now := time.Now().UTC()

	var expiresAt *time.Time
	if ttl != 0 {
		exp := now.Add(ttl)
		expiresAt = &exp
	}

	credID := fmt.Sprintf("cred_%s", hex.EncodeToString(rawBytes[:8]))

	cred := &Credential{
		ID:          credID,
		TenantID:    tenantID,
		SubjectID:   subjectID,
		KeyPrefix:   rawKey[:13] + "...",
		HashedKey:   hash,
		Scopes:      scopes,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
		Revoked:     false,
		Description: description,
	}

	return rawKey, cred, nil
}

// HashKey computes the deterministic SHA-256 digest of a raw API key.
func HashKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

// ValidateKey checks whether the provided raw key matches the credential and is active.
func (c *Credential) ValidateKey(rawKey string) error {
	if c.Revoked {
		return errors.New("credential has been revoked")
	}
	if c.ExpiresAt != nil && time.Now().UTC().After(*c.ExpiresAt) {
		return errors.New("credential has expired")
	}
	if HashKey(rawKey) != c.HashedKey {
		return errors.New("invalid key")
	}
	return nil
}

// HasScope checks if the credential holds the required scope (or ScopeAdmin).
func (c *Credential) HasScope(requiredScope string) bool {
	for _, s := range c.Scopes {
		if s == ScopeAdmin || s == requiredScope {
			return true
		}
		// Wildcard match e.g. "agents:*"
		if strings.HasSuffix(s, ":*") && strings.HasPrefix(requiredScope, strings.TrimSuffix(s, "*")) {
			return true
		}
	}
	return false
}

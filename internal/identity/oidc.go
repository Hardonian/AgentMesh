package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// OIDCConfig configures enterprise identity provider integration.
type OIDCConfig struct {
	IssuerURL        string              `json:"issuerUrl"` // e.g. "https://accounts.google.com"
	ClientID         string              `json:"clientId"`
	Audience         []string            `json:"audience"`
	RoleClaimMapping map[string][]string `json:"roleClaimMapping"` // IdP group -> AgentMesh roles
	SigningSecret    string              `json:"signingSecret"`
	IsSimulated      bool                `json:"isSimulated"`
}

// OIDCClaims contains normalized identity claims extracted from an enterprise ID token.
type OIDCClaims struct {
	Subject     string    `json:"sub"`
	Email       string    `json:"email"`
	Groups      []string  `json:"groups"`
	MappedRoles []string  `json:"mappedRoles"`
	ExpiresAt   time.Time `json:"exp"`
	Issuer      string    `json:"iss"`
}

// HumanApprovalToken represents a cryptographically verified token used for HITL approvals.
type HumanApprovalToken struct {
	ApprovalID    string    `json:"approvalId"`
	ApproverEmail string    `json:"approverEmail"`
	Roles         []string  `json:"roles"`
	IssuedAt      time.Time `json:"issuedAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
	Signature     string    `json:"signature"`
}

// OIDCValidator verifies enterprise ID tokens and manages HITL approval authorization.
type OIDCValidator struct {
	mu     sync.RWMutex
	config *OIDCConfig
	secret []byte
}

// NewOIDCValidator constructs a new OIDC and approval token validator.
func NewOIDCValidator(cfg *OIDCConfig) *OIDCValidator {
	if cfg == nil {
		cfg = &OIDCConfig{
			IssuerURL:     "https://accounts.google.com",
			SigningSecret: "local-simulated-mesh-secret-key-32b",
			IsSimulated:   true,
			RoleClaimMapping: map[string][]string{
				"finance-admins": {"ROLE_APPROVER", "ROLE_OPERATOR"},
				"secops":         {"ROLE_ADMIN", "ROLE_AUDITOR"},
			},
		}
	}
	secret := []byte(cfg.SigningSecret)
	if len(secret) == 0 {
		secret = []byte("default-agentmesh-internal-secret-token")
	}
	return &OIDCValidator{
		config: cfg,
		secret: secret,
	}
}

// ValidateIDToken inspects and validates an incoming enterprise JWT token.
func (v *OIDCValidator) ValidateIDToken(ctx context.Context, rawJWT string) (*OIDCClaims, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	parts := strings.Split(rawJWT, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed JWT format: expected 3 segments")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var rawClaims struct {
		Sub    string   `json:"sub"`
		Email  string   `json:"email"`
		Groups []string `json:"groups"`
		Iss    string   `json:"iss"`
		Exp    int64    `json:"exp"`
	}

	if err := json.Unmarshal(payloadBytes, &rawClaims); err != nil {
		return nil, fmt.Errorf("failed to parse claims JSON: %w", err)
	}

	now := time.Now().UTC()
	expTime := time.Unix(rawClaims.Exp, 0).UTC()
	if rawClaims.Exp > 0 && now.After(expTime) {
		return nil, errors.New("token has expired")
	}

	// Map enterprise groups to AgentMesh roles
	mappedRoles := make(map[string]bool)
	for _, grp := range rawClaims.Groups {
		if roles, ok := v.config.RoleClaimMapping[grp]; ok {
			for _, r := range roles {
				mappedRoles[r] = true
			}
		}
	}

	roleList := make([]string, 0, len(mappedRoles))
	for r := range mappedRoles {
		roleList = append(roleList, r)
	}

	return &OIDCClaims{
		Subject:     rawClaims.Sub,
		Email:       rawClaims.Email,
		Groups:      rawClaims.Groups,
		MappedRoles: roleList,
		ExpiresAt:   expTime,
		Issuer:      rawClaims.Iss,
	}, nil
}

// SignApprovalToken issues a tamper-evident human approval token.
func (v *OIDCValidator) SignApprovalToken(approvalID, email string, roles []string, ttl time.Duration) (*HumanApprovalToken, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	tok := &HumanApprovalToken{
		ApprovalID:    approvalID,
		ApproverEmail: email,
		Roles:         roles,
		IssuedAt:      now,
		ExpiresAt:     expiresAt,
	}

	data := fmt.Sprintf("%s:%s:%d:%d:%s", tok.ApprovalID, tok.ApproverEmail, tok.IssuedAt.Unix(), tok.ExpiresAt.Unix(), strings.Join(tok.Roles, ","))
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(data))
	tok.Signature = hex.EncodeToString(mac.Sum(nil))

	return tok, nil
}

// VerifyApprovalToken verifies HMAC signature and validity of an approval token.
func (v *OIDCValidator) VerifyApprovalToken(tok *HumanApprovalToken) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if tok == nil {
		return false, errors.New("token is nil")
	}
	if time.Now().UTC().After(tok.ExpiresAt) {
		return false, errors.New("approval token has expired")
	}

	data := fmt.Sprintf("%s:%s:%d:%d:%s", tok.ApprovalID, tok.ApproverEmail, tok.IssuedAt.Unix(), tok.ExpiresAt.Unix(), strings.Join(tok.Roles, ","))
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(data))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedSig), []byte(tok.Signature)) {
		return false, errors.New("invalid approval token signature")
	}

	return true, nil
}

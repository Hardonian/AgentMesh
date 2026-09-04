package mutation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

var (
	ErrRouteNotFound         = errors.New("route specification not found")
	ErrNoPriorConfig         = errors.New("no prior signed configuration to rollback to")
	ErrSignatureVerification = errors.New("signed config verification failed")
)

// SignedRouteConfig is an immutable, cryptographically verified data-plane route configuration.
type SignedRouteConfig struct {
	ConfigID           string            `json:"configId"`
	SequenceVersion    int64             `json:"sequenceVersion"`
	OrganizationID     string            `json:"organizationId"`
	CapabilityID       string            `json:"capabilityId"`
	Routes             map[string]int    `json:"routes"` // AgentID -> Weight %
	Fallbacks          []string          `json:"fallbacks"`
	PinnedAgent        string            `json:"pinnedAgent,omitempty"`
	PolicyHash         string            `json:"policyHash"`
	PayloadHash        string            `json:"payloadHash"`
	PreviousConfigHash string            `json:"previousConfigHash"`
	Signature          string            `json:"signature"`
	IssuedAt           time.Time         `json:"issuedAt"`
	EffectiveAt        time.Time         `json:"effectiveAt"`
}

// RouteMutator coordinates route state transitions and signed config distribution.
type RouteMutator struct {
	mu           sync.RWMutex
	signingKey   string
	activeSpecs  map[string]*spec.AgentRoutingSpec // CapabilityID -> Spec
	configChain  map[string][]*SignedRouteConfig   // CapabilityID -> History of signed configs
	lastKnownGood map[string]*spec.LastKnownGoodRoute // CapabilityID -> LastKnownGood
}

// NewRouteMutator creates a new route mutator with a signing secret.
func NewRouteMutator(signingKey string) *RouteMutator {
	if signingKey == "" {
		signingKey = "agentmesh-default-route-mutation-secret-key-32b"
	}
	return &RouteMutator{
		signingKey:    signingKey,
		activeSpecs:   make(map[string]*spec.AgentRoutingSpec),
		configChain:   make(map[string][]*SignedRouteConfig),
		lastKnownGood: make(map[string]*spec.LastKnownGoodRoute),
	}
}

// RegisterRoute initializes a routing specification.
func (m *RouteMutator) RegisterRoute(s *spec.AgentRoutingSpec) (*SignedRouteConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.activeSpecs[s.CapabilityID] = s
	return m.generateSignedConfigLocked(s, "")
}

// ChangeRouteWeights adjusts candidate weights and generates a new signed config.
func (m *RouteMutator) ChangeRouteWeights(capabilityID string, newWeights map[string]int, policyHash string) (*SignedRouteConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rs, exists := m.activeSpecs[capabilityID]
	if !exists {
		return nil, ErrRouteNotFound
	}

	rs.Weights = newWeights
	rs.UpdatedAt = time.Now().UTC()
	return m.generateSignedConfigLocked(rs, policyHash)
}

// PromoteCandidate sets a candidate agent to 100% traffic and records it as last known good.
func (m *RouteMutator) PromoteCandidate(capabilityID, candidateAgent string, policyHash string) (*SignedRouteConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rs, exists := m.activeSpecs[capabilityID]
	if !exists {
		return nil, ErrRouteNotFound
	}

	newWeights := make(map[string]int)
	for agent := range rs.Weights {
		newWeights[agent] = 0
	}
	newWeights[candidateAgent] = 100
	rs.Weights = newWeights
	rs.UpdatedAt = time.Now().UTC()

	cfg, err := m.generateSignedConfigLocked(rs, policyHash)
	if err != nil {
		return nil, err
	}

	// Record Last Known Good
	m.lastKnownGood[capabilityID] = &spec.LastKnownGoodRoute{
		CapabilityID:        capabilityID,
		OrganizationID:      rs.OrganizationID,
		ProjectID:           rs.ProjectID,
		RouteSpecHash:       rs.ComputeSpecHash(),
		AgentID:             candidateAgent,
		ObservationWindowMs: 60000,
		SuccessRate:         1.0,
		QualifiedAt:         time.Now().UTC(),
		VerifiedBy:          "CANARY_PROMOTION",
	}

	return cfg, nil
}

// RestorePriorRoute performs a single-action rollback to the previous signed configuration.
func (m *RouteMutator) RestorePriorRoute(capabilityID string) (*SignedRouteConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	history, exists := m.configChain[capabilityID]
	if !exists || len(history) < 2 {
		return nil, ErrNoPriorConfig
	}

	// Prior config is second to last
	prior := history[len(history)-2]

	rs, exists := m.activeSpecs[capabilityID]
	if !exists {
		return nil, ErrRouteNotFound
	}

	rs.Weights = prior.Routes
	rs.UpdatedAt = time.Now().UTC()

	return m.generateSignedConfigLocked(rs, prior.PolicyHash)
}

// VerifyConfigSignature validates that a signed route configuration is authentic and untampered.
func (m *RouteMutator) VerifyConfigSignature(cfg *SignedRouteConfig) (bool, error) {
	expectedPayloadHash := computePayloadHash(cfg.CapabilityID, cfg.Routes, cfg.Fallbacks, cfg.SequenceVersion)
	if expectedPayloadHash != cfg.PayloadHash {
		return false, ErrSignatureVerification
	}

	expectedSig := computeHMACSignature(cfg.PayloadHash, cfg.PreviousConfigHash, m.signingKey)
	if !hmac.Equal([]byte(expectedSig), []byte(cfg.Signature)) {
		return false, ErrSignatureVerification
	}
	return true, nil
}

func (m *RouteMutator) generateSignedConfigLocked(rs *spec.AgentRoutingSpec, policyHash string) (*SignedRouteConfig, error) {
	now := time.Now().UTC()
	history := m.configChain[rs.CapabilityID]

	var seq int64 = 1
	var prevHash string = "genesis_root_hash"
	if len(history) > 0 {
		latest := history[len(history)-1]
		seq = latest.SequenceVersion + 1
		prevHash = latest.Signature
	}

	cfgID := fmt.Sprintf("cfg_%s_v%d_%d", rs.CapabilityID, seq, now.Unix())
	payloadHash := computePayloadHash(rs.CapabilityID, rs.Weights, rs.Fallbacks, seq)
	sig := computeHMACSignature(payloadHash, prevHash, m.signingKey)

	cfg := &SignedRouteConfig{
		ConfigID:           cfgID,
		SequenceVersion:    seq,
		OrganizationID:     rs.OrganizationID,
		CapabilityID:       rs.CapabilityID,
		Routes:             rs.Weights,
		Fallbacks:          rs.Fallbacks,
		PolicyHash:         policyHash,
		PayloadHash:        payloadHash,
		PreviousConfigHash: prevHash,
		Signature:          sig,
		IssuedAt:           now,
		EffectiveAt:        now,
	}

	m.configChain[rs.CapabilityID] = append(m.configChain[rs.CapabilityID], cfg)
	return cfg, nil
}

func computePayloadHash(capabilityID string, routes map[string]int, fallbacks []string, seq int64) string {
	h := sha256.New()
	content := fmt.Sprintf("%s:%v:%v:%d", capabilityID, routes, fallbacks, seq)
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

func computeHMACSignature(payloadHash, prevHash, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(payloadHash + ":" + prevHash))
	return hex.EncodeToString(mac.Sum(nil))
}

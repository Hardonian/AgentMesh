package approval

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "PENDING"
	StatusApproved Status = "APPROVED"
	StatusRejected Status = "REJECTED"
	StatusExpired  Status = "EXPIRED"
	StatusCanceled Status = "CANCELED"
)

var (
	ErrApprovalNotFound       = errors.New("approval request not found")
	ErrApprovalNotPending     = errors.New("approval is not in PENDING state")
	ErrApprovalExpired        = errors.New("approval request has expired")
	ErrApprovalTampered       = errors.New("approval token parameter mismatch: parameters have been tampered with")
	ErrApprovalUnauthorized   = errors.New("approver is not authorized for this request")
	ErrApprovalConsumed       = errors.New("approval token has already been consumed")
	ErrApprovalTenantMismatch = errors.New("approval token was issued for a different tenant")
)

// Request encapsulates a pending human approval decision.
type Request struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenantId"`
	AgentID        string         `json:"agentId"`
	Tool           string         `json:"tool"`
	Action         string         `json:"action"`
	Parameters     map[string]any `json:"parameters"`
	ParametersHash string         `json:"parametersHash"`
	PolicyID       string         `json:"policyId"`
	PolicyVersion  string         `json:"policyVersion"`
	Reason         string         `json:"reason"`
	Status         Status         `json:"status"`
	CreatedAt      time.Time      `json:"createdAt"`
	ExpiresAt      time.Time      `json:"expiresAt"`
	ResolvedAt     *time.Time     `json:"resolvedAt,omitempty"`
	ResolvedBy     string         `json:"resolvedBy,omitempty"`
	ResolutionNote string         `json:"resolutionNote,omitempty"`
	ApprovalToken  string         `json:"approvalToken,omitempty"` // Nonce returned to agent
	Consumed       bool           `json:"consumed"`
}

// Service manages the lifecycle of approval requests.
type Service struct {
	mu       sync.RWMutex
	requests map[string]*Request
}

// NewService constructs an approval service.
func NewService() *Service {
	return &Service{
		requests: make(map[string]*Request),
	}
}

// ComputeParametersHash produces a deterministic SHA-256 hash of arbitrary tool parameters.
func ComputeParametersHash(params map[string]any) (string, error) {
	if params == nil {
		params = make(map[string]any)
	}
	bytes, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(bytes)
	return hex.EncodeToString(h[:]), nil
}

// CreateRequest registers a new pending approval requirement.
func (s *Service) CreateRequest(tenantID, agentID, tool, action string, params map[string]any, policyID, policyVersion, reason string, ttl time.Duration) (*Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	paramHash, err := ComputeParametersHash(params)
	if err != nil {
		return nil, fmt.Errorf("failed to hash parameters: %w", err)
	}

	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	now := time.Now().UTC()
	reqID := "appr_" + uuid.NewString()[:12]

	req := &Request{
		ID:             reqID,
		TenantID:       tenantID,
		AgentID:        agentID,
		Tool:           tool,
		Action:         action,
		Parameters:     params,
		ParametersHash: paramHash,
		PolicyID:       policyID,
		PolicyVersion:  policyVersion,
		Reason:         reason,
		Status:         StatusPending,
		CreatedAt:      now,
		ExpiresAt:      now.Add(ttl),
	}

	s.requests[reqID] = req
	return req, nil
}

// Resolve processes an approval or rejection by a human reviewer.
func (s *Service) Resolve(requestID, reviewerID string, approve bool, note string) (*Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req, exists := s.requests[requestID]
	if !exists {
		return nil, ErrApprovalNotFound
	}

	now := time.Now().UTC()
	if now.After(req.ExpiresAt) {
		req.Status = StatusExpired
		return nil, ErrApprovalExpired
	}

	if req.Status != StatusPending {
		return nil, fmt.Errorf("%w: current status %s", ErrApprovalNotPending, req.Status)
	}

	req.ResolvedAt = &now
	req.ResolvedBy = reviewerID
	req.ResolutionNote = note

	if approve {
		req.Status = StatusApproved
		// Generate an approval token bound to exact agent, tool, and parameters hash
		tokenRaw := fmt.Sprintf("%s|%s|%s|%s|%d", req.ID, req.AgentID, req.Tool, req.ParametersHash, now.Unix())
		h := sha256.Sum256([]byte(tokenRaw))
		req.ApprovalToken = "tok_" + hex.EncodeToString(h[:16])
	} else {
		req.Status = StatusRejected
	}

	return req, nil
}

// ValidateApproval verifies that a presented approval token is valid and matches the exact parameters.
func (s *Service) ValidateApproval(requestID, agentID, tool string, params map[string]any, token string) error {
	return s.ValidateApprovalForTenant(requestID, "", agentID, tool, params, token)
}

// ValidateApprovalForTenant verifies that a presented approval token is valid, unexpired, unconsumed, and matches the exact parameters and tenant.
func (s *Service) ValidateApprovalForTenant(requestID, tenantID, agentID, tool string, params map[string]any, token string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.validateInternal(requestID, tenantID, agentID, tool, params, token)
}

func (s *Service) validateInternal(requestID, tenantID, agentID, tool string, params map[string]any, token string) error {
	req, exists := s.requests[requestID]
	if !exists {
		return ErrApprovalNotFound
	}

	if tenantID != "" && req.TenantID != "" && req.TenantID != tenantID {
		return ErrApprovalTenantMismatch
	}

	now := time.Now().UTC()
	if now.After(req.ExpiresAt) {
		req.Status = StatusExpired
		return ErrApprovalExpired
	}

	if req.Consumed {
		return ErrApprovalConsumed
	}

	if req.Status != StatusApproved {
		return fmt.Errorf("request is not approved: %s", req.Status)
	}

	if subtle.ConstantTimeCompare([]byte(req.ApprovalToken), []byte(token)) != 1 {
		return errors.New("invalid approval token")
	}

	if req.AgentID != agentID || req.Tool != tool {
		return errors.New("approval token was issued for a different agent or tool")
	}

	// Critical Invariant: Parameter tampering check
	currentHash, err := ComputeParametersHash(params)
	if err != nil {
		return err
	}

	if currentHash != req.ParametersHash {
		return fmt.Errorf("%w: parameters do not match approved invocation (expected %s, got %s)",
			ErrApprovalTampered, req.ParametersHash, currentHash)
	}

	return nil
}

// ConsumeApproval validates the approval token and marks it consumed, preventing replay.
func (s *Service) ConsumeApproval(requestID, tenantID, agentID, tool string, params map[string]any, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.validateInternal(requestID, tenantID, agentID, tool, params, token); err != nil {
		return err
	}

	req := s.requests[requestID]
	req.Consumed = true
	return nil
}

// ListPending returns all pending approvals for a tenant.
func (s *Service) ListPending(tenantID string) []*Request {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*Request
	now := time.Now().UTC()
	for _, r := range s.requests {
		if tenantID != "" && r.TenantID != tenantID {
			continue
		}
		if r.Status == StatusPending {
			if now.After(r.ExpiresAt) {
				r.Status = StatusExpired
				continue
			}
			list = append(list, r)
		}
	}
	return list
}

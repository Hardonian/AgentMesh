package policy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// PolicyCanary governs side-by-side execution between baseline and candidate policy versions.
type PolicyCanary struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenantId"`
	BaselinePolicy     *Policy   `json:"baselinePolicy"`
	CandidatePolicy    *Policy   `json:"candidatePolicy"`
	LastKnownGood      *Policy   `json:"lastKnownGood"`
	ShadowMode         bool      `json:"shadowMode"` // Always true during evaluation
	EvaluatedTraffic   int64     `json:"evaluatedTraffic"`
	WouldAllowCount    int64     `json:"wouldAllowCount"`
	WouldDenyCount     int64     `json:"wouldDenyCount"`
	WouldApproveCount  int64     `json:"wouldApproveCount"`
	DiscrepancyCount   int64     `json:"discrepancyCount"`
	CreatedAt          time.Time `json:"createdAt"`
	PromotedAt         *time.Time `json:"promotedAt,omitempty"`
	RolledBackAt       *time.Time `json:"rolledBackAt,omitempty"`
}

// ShadowEvaluator coordinates live shadow evaluation without disrupting production traffic.
type ShadowEvaluator struct {
	mu           sync.RWMutex
	canaries     map[string]*PolicyCanary // canaryID -> canary
	baselineEng  *Engine
	candidateEng *Engine
}

// NewShadowEvaluator creates a shadow policy canary.
func NewShadowEvaluator(canary *PolicyCanary) *ShadowEvaluator {
	se := &ShadowEvaluator{
		canaries: make(map[string]*PolicyCanary),
	}
	if canary != nil {
		se.canaries[canary.ID] = canary
		se.baselineEng = NewEngine([]*Policy{canary.BaselinePolicy})
		se.candidateEng = NewEngine([]*Policy{canary.CandidatePolicy})
	}
	return se
}

// EvaluateShadow evaluates traffic: executes baseline as ENFORCED, while evaluating candidate in SHADOW.
// Invariant: Shadow evaluation NEVER enforces candidate decisions on live traffic!
func (se *ShadowEvaluator) EvaluateShadow(ctx context.Context, canaryID string, req *EvaluationRequest) (*Decision, *Decision, error) {
	se.mu.Lock()
	defer se.mu.Unlock()

	c, ok := se.canaries[canaryID]
	if !ok {
		return nil, nil, fmt.Errorf("policy canary %s not found", canaryID)
	}

	// 1. Enforced baseline decision
	baseDec := se.baselineEng.Evaluate(ctx, req)

	// 2. Shadow candidate decision
	candDec := se.candidateEng.Evaluate(ctx, req)

	c.EvaluatedTraffic++
	switch candDec.Effect {
	case EffectAllow:
		c.WouldAllowCount++
	case EffectDeny:
		c.WouldDenyCount++
	case EffectRequireApproval:
		c.WouldApproveCount++
	}

	if baseDec.Effect != candDec.Effect {
		c.DiscrepancyCount++
	}

	return baseDec, candDec, nil
}

// Promote activates the candidate policy as the new baseline and saves last known good.
func (se *ShadowEvaluator) Promote(canaryID string) (*Policy, error) {
	se.mu.Lock()
	defer se.mu.Unlock()

	c, ok := se.canaries[canaryID]
	if !ok {
		return nil, errors.New("canary not found")
	}

	now := time.Now().UTC()
	c.LastKnownGood = c.BaselinePolicy
	c.BaselinePolicy = c.CandidatePolicy
	c.ShadowMode = false
	c.PromotedAt = &now

	se.baselineEng = NewEngine([]*Policy{c.BaselinePolicy})
	return c.BaselinePolicy, nil
}

// Rollback restores the last known good policy immediately.
func (se *ShadowEvaluator) Rollback(canaryID string) (*Policy, error) {
	se.mu.Lock()
	defer se.mu.Unlock()

	c, ok := se.canaries[canaryID]
	if !ok {
		return nil, errors.New("canary not found")
	}
	if c.LastKnownGood == nil {
		return nil, errors.New("no last known good policy recorded for rollback")
	}

	now := time.Now().UTC()
	c.BaselinePolicy = c.LastKnownGood
	c.ShadowMode = false
	c.RolledBackAt = &now

	se.baselineEng = NewEngine([]*Policy{c.BaselinePolicy})
	return c.BaselinePolicy, nil
}

package passport

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

// EvidenceStatus distinguishes declared claims from empirical telemetry.
type EvidenceStatus string

const (
	EvidenceDeclared EvidenceStatus = "DECLARED"
	EvidenceMeasured EvidenceStatus = "MEASURED"
	EvidenceInferred EvidenceStatus = "INFERRED"
	EvidenceUnknown  EvidenceStatus = "UNKNOWN"
)

// PolicyStatus indicates current compliance status.
type PolicyCompliance string

const (
	ComplianceCompliant   PolicyCompliance = "COMPLIANT"
	ComplianceViolations  PolicyCompliance = "VIOLATIONS_DETECTED"
	ComplianceUnreviewed  PolicyCompliance = "UNREVIEWED"
)

// AgentPassport combines static declared configuration with operational evidence.
type AgentPassport struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	Identity   PassportIdentity `json:"identity"`
	Declared   DeclaredClaims   `json:"declared"`
	Operational MeasuredEvidence `json:"operational"`
	Audit      PassportAudit    `json:"audit"`
}

type PassportIdentity struct {
	AgentID      string `json:"agentId"`
	Organization string `json:"organization"`
	Version      string `json:"version"`
	Runtime      string `json:"runtime"`
	Framework    string `json:"framework"`
}

type DeclaredClaims struct {
	Capabilities []string `json:"capabilities"`
	Tools        []string `json:"tools"`
	Delegates    []string `json:"delegates"`
	TargetSLO    struct {
		SuccessRate  float64 `json:"successRate"`
		P95LatencyMs int64   `json:"p95LatencyMs"`
	} `json:"targetSlo"`
	ContractHash string `json:"contractHash"`
}

type MeasuredEvidence struct {
	Status              EvidenceStatus   `json:"status"` // MEASURED when sufficient runs exist
	SampleCount         int64            `json:"sampleCount"`
	SuccessRate         float64          `json:"successRate"`
	P50LatencyMs        int64            `json:"p50LatencyMs"`
	P95LatencyMs        int64            `json:"p95LatencyMs"`
	AverageCostUSD      float64          `json:"averageCostUsd"`
	ToolCallSuccessRate float64          `json:"toolCallSuccessRate"`
	PolicyCompliance    PolicyCompliance `json:"policyCompliance"`
	EvaluationScore     float64          `json:"evaluationScore"`
	RegressionsDetected int              `json:"regressionsDetected"`
	LastEvaluatedAt     *time.Time       `json:"lastEvaluatedAt,omitempty"`
	ConfidenceScore     float64          `json:"confidenceScore"` // 0.0 to 1.0 based on sample size
}

type PassportAudit struct {
	IssuedAt  time.Time `json:"issuedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Issuer    string    `json:"issuer"`
}

// GenerateFromContract initializes a passport from a contract with DECLARED status.
func GenerateFromContract(c *contracts.AgentContract, runtime, framework string) (*AgentPassport, error) {
	if c == nil {
		return nil, errors.New("contract cannot be nil")
	}

	hash, err := c.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to hash contract: %w", err)
	}

	now := time.Now().UTC()
	pass := &AgentPassport{
		APIVersion: "agentmesh.dev/v1",
		Kind:       "AgentPassport",
		Identity: PassportIdentity{
			AgentID:      c.Metadata.Name,
			Organization: c.Metadata.Organization,
			Version:      c.Metadata.Version,
			Runtime:      runtime,
			Framework:    framework,
		},
		Declared: DeclaredClaims{
			Capabilities: c.Capabilities,
			Tools:        c.Tools.Allow,
			Delegates:    c.Delegation.Allow,
			ContractHash: hash,
		},
		Operational: MeasuredEvidence{
			Status:           EvidenceDeclared, // Initially declared, not yet measured
			PolicyCompliance: ComplianceCompliant,
			ConfidenceScore:  0.0,
		},
		Audit: PassportAudit{
			IssuedAt:  now,
			ExpiresAt: now.Add(24 * 30 * time.Hour), // 30 days
			Issuer:    "AgentMesh Control Plane",
		},
	}
	pass.Declared.TargetSLO.SuccessRate = c.SLO.SuccessRate
	pass.Declared.TargetSLO.P95LatencyMs = c.SLO.P95LatencyMs
	return pass, nil
}

// RecordExecutionSample updates operational evidence with a completed task sample.
func (p *AgentPassport) RecordExecutionSample(success bool, latencyMs int64, costUSD float64, toolCallSuccess bool) {
	ev := &p.Operational
	prevCount := float64(ev.SampleCount)
	newCount := prevCount + 1
	ev.SampleCount = int64(newCount)

	// Rolling averages
	var succFloat float64
	if success {
		succFloat = 1.0
	}
	ev.SuccessRate = (ev.SuccessRate*prevCount + succFloat) / newCount
	ev.AverageCostUSD = (ev.AverageCostUSD*prevCount + costUSD) / newCount

	var toolSuccFloat float64
	if toolCallSuccess {
		toolSuccFloat = 1.0
	}
	ev.ToolCallSuccessRate = (ev.ToolCallSuccessRate*prevCount + toolSuccFloat) / newCount

	// Latency approximation update
	if ev.P95LatencyMs == 0 || latencyMs > ev.P95LatencyMs {
		ev.P95LatencyMs = latencyMs
	}

	// Upgrade status to MEASURED once threshold reached
	if ev.SampleCount >= 5 {
		ev.Status = EvidenceMeasured
	} else {
		ev.Status = EvidenceInferred
	}

	// Confidence scales with sample count up to 100 samples
	if ev.SampleCount >= 100 {
		ev.ConfidenceScore = 1.0
	} else {
		ev.ConfidenceScore = float64(ev.SampleCount) / 100.0
	}
}

// ToJSON exports passport to JSON.
func (p *AgentPassport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// FromJSON decodes a passport from JSON.
func FromJSON(data []byte) (*AgentPassport, error) {
	var p AgentPassport
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse passport JSON: %w", err)
	}
	return &p, nil
}

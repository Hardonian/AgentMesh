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
	EvidenceObserved EvidenceStatus = "OBSERVED"
	EvidenceInferred EvidenceStatus = "INFERRED"
	EvidenceUnknown  EvidenceStatus = "UNKNOWN"
)

// PolicyCompliance indicates current policy compliance status.
type PolicyCompliance string

const (
	ComplianceCompliant  PolicyCompliance = "COMPLIANT"
	ComplianceViolations PolicyCompliance = "VIOLATIONS_DETECTED"
	ComplianceUnreviewed PolicyCompliance = "UNREVIEWED"
)

// AgentPassport V2 combines declared configuration with deep empirical operational evidence and provenance.
type AgentPassport struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	Identity      PassportIdentity      `json:"identity"`
	Protocol      ProtocolProfile       `json:"protocol"`
	Graph         GraphSummary          `json:"graph"`
	Compatibility CompatibilityProfile  `json:"compatibility"`
	Reliability   ReliabilityMetrics    `json:"reliability"`
	Economics     EconomicMetrics       `json:"economics"`
	Safety        SafetyProfile         `json:"safety"`
	Evaluations   EvaluationSummary     `json:"evaluations"`
	Deployment    DeploymentState       `json:"deployment"`
	Provenance    map[string]Provenance `json:"provenance"` // Keyed by section name
	Audit         PassportAudit         `json:"audit"`
	IsPublic      bool                  `json:"isPublic"`
}

type Provenance struct {
	Tier       EvidenceStatus `json:"tier"`
	Source     string         `json:"source"`
	Confidence float64        `json:"confidence"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type PassportIdentity struct {
	AgentID      string `json:"agentId"`
	Organization string `json:"organization"`
	Version      string `json:"version"`
	Runtime      string `json:"runtime"`   // e.g. "go", "python"
	Framework    string `json:"framework"` // e.g. "google-adk", "langgraph", "custom"
}

type ProtocolProfile struct {
	A2AVersion string   `json:"a2aVersion"`
	MCPSupport bool     `json:"mcpSupport"`
	Protocols  []string `json:"protocols"` // ["a2a", "mcp"]
}

type GraphSummary struct {
	GraphHash      string   `json:"graphHash,omitempty"`
	NodeCount      int      `json:"nodeCount"`
	Delegates      []string `json:"delegates"`
	Tools          []string `json:"tools"`
	ApprovalPoints []string `json:"approvalPoints"`
}

type CompatibilityProfile struct {
	A2AStatus string `json:"a2aStatus"` // COMPATIBLE, PARTIALLY_COMPATIBLE, INCOMPATIBLE, UNTESTED
	MCPStatus string `json:"mcpStatus"`
	MatrixRef string `json:"matrixRef,omitempty"`
}

type ReliabilityMetrics struct {
	TaskSuccessRate     float64 `json:"taskSuccessRate"`
	ToolCallSuccessRate float64 `json:"toolCallSuccessRate"`
	P50LatencyMs        int64   `json:"p50LatencyMs"`
	P95LatencyMs        int64   `json:"p95LatencyMs"`
	TimeoutRate         float64 `json:"timeoutRate"`
	RetryRate           float64 `json:"retryRate"`
	SampleCount         int64   `json:"sampleCount"`
	RouteSharePct       float64 `json:"routeSharePct,omitempty"`
	FailoverCount       int64   `json:"failoverCount,omitempty"`
	SLOStatus           string  `json:"sloStatus,omitempty"`
}

type EconomicMetrics struct {
	AverageCostUSD  float64 `json:"averageCostUsd"`
	TokensPerTask   int64   `json:"tokensPerTask"`
	DailyCostUSD    float64 `json:"dailyCostUsd"`
	MaxObservedCost float64 `json:"maxObservedCost"`
}

type SafetyProfile struct {
	PolicyCoverage       string           `json:"policyCoverage"` // FULLY_GOVERNED, PARTIAL, UNGOVERNED
	PolicyCompliance     PolicyCompliance `json:"policyCompliance"`
	ApprovalRequired     bool             `json:"approvalRequired"`
	PrivilegedToolCount  int              `json:"privilegedToolCount"`
	DelegationDepthLimit int              `json:"delegationDepthLimit"`
	RiskScore            string           `json:"riskScore"` // LOW, MEDIUM, HIGH, CRITICAL
}

type EvaluationSummary struct {
	LatestScore         float64    `json:"latestScore"`
	BaselineScore       float64    `json:"baselineScore"`
	RegressionsDetected int        `json:"regressionsDetected"`
	LastEvaluatedAt     *time.Time `json:"lastEvaluatedAt,omitempty"`
	SuiteID             string     `json:"suiteId,omitempty"`
	Freshness           string     `json:"freshness,omitempty"` // CURRENT, AGING, STALE, INVALIDATED
}

type DeploymentState struct {
	CurrentVersion       string `json:"currentVersion"`
	CandidateVersion     string `json:"candidateVersion,omitempty"`
	LastKnownGoodVersion string `json:"lastKnownGoodVersion"`
	Status               string `json:"status"` // ACTIVE, CANARY, DEGRADED, DISABLED
}

type PassportAudit struct {
	IssuedAt     time.Time `json:"issuedAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
	Issuer       string    `json:"issuer"`
	ContractHash string    `json:"contractHash"`
}

// GenerateFromContract initializes a Passport V2 from an AgentContract with explicit DECLARED provenance.
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
		APIVersion: "agentmesh.dev/v2alpha1",
		Kind:       "AgentPassport",
		Identity: PassportIdentity{
			AgentID:      c.Metadata.Name,
			Organization: c.Metadata.Organization,
			Version:      c.Metadata.Version,
			Runtime:      runtime,
			Framework:    framework,
		},
		Protocol: ProtocolProfile{
			A2AVersion: "0.3.0",
			MCPSupport: true,
			Protocols:  c.Identity.Protocols,
		},
		Graph: GraphSummary{
			NodeCount:      len(c.Tools.Allow) + len(c.Delegation.Allow) + 1,
			Delegates:      c.Delegation.Allow,
			Tools:          c.Tools.Allow,
			ApprovalPoints: c.Approval.RequiredFor,
		},
		Compatibility: CompatibilityProfile{
			A2AStatus: "UNTESTED",
			MCPStatus: "GOVERNED",
		},
		Reliability: ReliabilityMetrics{
			TaskSuccessRate:     0.0,
			ToolCallSuccessRate: 0.0,
			P95LatencyMs:        c.SLO.P95LatencyMs,
			SampleCount:         0,
		},
		Economics: EconomicMetrics{
			AverageCostUSD: 0.0,
			TokensPerTask:  0,
		},
		Safety: SafetyProfile{
			PolicyCoverage:       "FULLY_GOVERNED",
			PolicyCompliance:     ComplianceCompliant,
			ApprovalRequired:     len(c.Approval.RequiredFor) > 0,
			DelegationDepthLimit: c.Delegation.MaxDepth,
			RiskScore:            "LOW",
		},
		Evaluations: EvaluationSummary{
			LatestScore:   1.0,
			BaselineScore: 1.0,
		},
		Deployment: DeploymentState{
			CurrentVersion:       c.Metadata.Version,
			LastKnownGoodVersion: c.Metadata.Version,
			Status:               "ACTIVE",
		},
		Provenance: map[string]Provenance{
			"identity":      {Tier: EvidenceDeclared, Source: "AgentContract", Confidence: 1.0, UpdatedAt: now},
			"reliability":   {Tier: EvidenceDeclared, Source: "SLO Target", Confidence: 0.0, UpdatedAt: now},
			"economics":     {Tier: EvidenceDeclared, Source: "Contract Ceiling", Confidence: 0.0, UpdatedAt: now},
			"safety":        {Tier: EvidenceDeclared, Source: "Policy Declarations", Confidence: 1.0, UpdatedAt: now},
			"compatibility": {Tier: EvidenceInferred, Source: "Declared Protocol", Confidence: 0.5, UpdatedAt: now},
		},
		Audit: PassportAudit{
			IssuedAt:     now,
			ExpiresAt:    now.Add(24 * 30 * time.Hour), // 30 days
			Issuer:       "AgentMesh Control Plane",
			ContractHash: hash,
		},
		IsPublic: false, // Strict default: private
	}

	return pass, nil
}

// RecordExecutionSample updates operational evidence with an empirical run.
func (p *AgentPassport) RecordExecutionSample(success bool, latencyMs int64, costUSD float64, toolCallSuccess bool) {
	ev := &p.Reliability
	prevCount := float64(ev.SampleCount)
	newCount := prevCount + 1
	ev.SampleCount = int64(newCount)

	var succFloat float64
	if success {
		succFloat = 1.0
	}
	ev.TaskSuccessRate = (ev.TaskSuccessRate*prevCount + succFloat) / newCount

	var toolSuccFloat float64
	if toolCallSuccess {
		toolSuccFloat = 1.0
	}
	ev.ToolCallSuccessRate = (ev.ToolCallSuccessRate*prevCount + toolSuccFloat) / newCount

	if ev.P95LatencyMs == 0 || latencyMs > ev.P95LatencyMs {
		ev.P95LatencyMs = latencyMs
	}

	p.Economics.AverageCostUSD = (p.Economics.AverageCostUSD*prevCount + costUSD) / newCount
	if costUSD > p.Economics.MaxObservedCost {
		p.Economics.MaxObservedCost = costUSD
	}

	now := time.Now().UTC()
	tier := EvidenceInferred
	if ev.SampleCount >= 5 {
		tier = EvidenceMeasured
	}
	confidence := float64(ev.SampleCount) / 100.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	p.Provenance["reliability"] = Provenance{
		Tier:       tier,
		Source:     "Production Telemetry",
		Confidence: confidence,
		UpdatedAt:  now,
	}
	p.Provenance["economics"] = Provenance{
		Tier:       tier,
		Source:     "Production Telemetry",
		Confidence: confidence,
		UpdatedAt:  now,
	}
}

// SanitizeForPublic returns a scrubbed copy safe for public resolution (no private endpoints, tools, or tenant IDs).
func (p *AgentPassport) SanitizeForPublic() *AgentPassport {
	if !p.IsPublic {
		return nil
	}
	clone := *p
	// 1. Redact organization / internal tenant ID
	clone.Identity.Organization = "[REDACTED_ORGANIZATION]"

	// 2. Redact private tools and delegates
	clone.Graph.Tools = []string{fmt.Sprintf("%d governed tools", len(p.Graph.Tools))}
	clone.Graph.Delegates = []string{fmt.Sprintf("%d authorized peer agents", len(p.Graph.Delegates))}
	clone.Graph.ApprovalPoints = []string{fmt.Sprintf("%d human approval gates", len(p.Graph.ApprovalPoints))}

	// 3. Redact private internal cost contract
	clone.Economics = EconomicMetrics{
		AverageCostUSD:  0,
		TokensPerTask:   0,
		DailyCostUSD:    0,
		MaxObservedCost: 0,
	}

	return &clone
}

// GenerateBadge returns an ASCII/SVG-ready status badge representation.
func (p *AgentPassport) GenerateBadge() string {
	status := p.Deployment.Status
	reliability := fmt.Sprintf("%.1f%%", p.Reliability.TaskSuccessRate*100)
	if p.Reliability.SampleCount == 0 {
		reliability = "UNTESTED"
	}
	return fmt.Sprintf("[AgentMesh|Passport: %s | %s | %s]", p.Identity.AgentID, status, reliability)
}

// ToJSON exports passport to indented JSON.
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

// RouterPassport represents operational provenance and verification of an AgentMesh routing algorithm.
type RouterPassport struct {
	APIVersion       string    `json:"apiVersion"`
	Kind             string    `json:"kind"`
	RouterID         string    `json:"routerId"`
	Version          string    `json:"version"`
	AlgorithmType    string    `json:"algorithmType"` // DETERMINISTIC_BASELINE, LEARNED_GBDT, CONTEXTUAL_BANDIT
	Objective        string    `json:"objective"`
	Status           string    `json:"status"` // CANDIDATE, SHADOW, ACTIVE, RETIRED
	AgreementRate    float64   `json:"agreementRate"`
	CostReductionPct float64   `json:"costReductionPct"`
	IssuedAt         time.Time `json:"issuedAt"`
	SignedBy         string    `json:"signedBy"`
}

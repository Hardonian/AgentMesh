package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// AgentRoutingSpec defines declarative desired state for capability routing.
type AgentRoutingSpec struct {
	CapabilityID       string            `json:"capabilityId"`
	OrganizationID     string            `json:"organizationId"`
	ProjectID          string            `json:"projectId"`
	EligibleAgents     []string          `json:"eligibleAgents"`
	PreferredAgents    []string          `json:"preferredAgents"`
	Weights            map[string]int    `json:"weights"` // AgentID -> Traffic weight %
	Fallbacks          []string          `json:"fallbacks"`
	MinimumReliability float64           `json:"minimumReliability"`
	MaximumCostUSD     float64           `json:"maximumCostUsd"`
	MaximumLatencyMs   int64             `json:"maximumLatencyMs"`
	MinimumQuality     float64           `json:"minimumQuality"`
	ModelConstraints   []string          `json:"modelConstraints,omitempty"`
	ToolRequirements   []string          `json:"toolRequirements,omitempty"`
	DataClassification string            `json:"dataClassification,omitempty"`
	RegionConstraints  []string          `json:"regionConstraints,omitempty"`
	ExplorationPolicy  string            `json:"explorationPolicy,omitempty"` // SAFE_EXPLORATION, OFF
	CanaryPolicy       string            `json:"canaryPolicy,omitempty"`
	Version            string            `json:"version"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

// ComputeSpecHash computes an immutable hash of the routing specification.
func (s *AgentRoutingSpec) ComputeSpecHash() string {
	h := sha256.New()
	content := fmt.Sprintf("%s:%s:%s:%v:%v:%v:%v:%.4f:%.4f:%d",
		s.OrganizationID,
		s.CapabilityID,
		s.Version,
		s.EligibleAgents,
		s.PreferredAgents,
		s.Weights,
		s.Fallbacks,
		s.MinimumReliability,
		s.MaximumCostUSD,
		s.MaximumLatencyMs,
	)
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

// LastKnownGoodRoute records a validated production route that has proven stability.
type LastKnownGoodRoute struct {
	CapabilityID        string         `json:"capabilityId"`
	OrganizationID      string         `json:"organizationId"`
	ProjectID           string         `json:"projectId"`
	RouteSpecHash       string         `json:"routeSpecHash"`
	AgentID             string         `json:"agentId"`
	AgentVersion        string         `json:"agentVersion"`
	ModelTarget         string         `json:"modelTarget"`
	ObservationWindowMs int64          `json:"observationWindowMs"`
	SampleCount         int64          `json:"sampleCount"`
	SuccessRate         float64        `json:"successRate"`
	P95LatencyMs        int64          `json:"p95LatencyMs"`
	CostPerTaskUSD      float64        `json:"costPerTaskUsd"`
	QualifiedAt         time.Time      `json:"qualifiedAt"`
	VerifiedBy          string         `json:"verifiedBy"` // SYSTEM_CANARY, OPERATOR
}

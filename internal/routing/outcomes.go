package routing

import (
	"errors"
	"time"

	"github.com/agentmesh/agentmesh/pkg/task"
)

// FailureType normalizes execution errors across agents, models, tools, and policies.
type FailureType string

const (
	FailureNone            FailureType = "NONE"
	FailureTimeout         FailureType = "TIMEOUT"
	FailureAgentError      FailureType = "AGENT_ERROR"
	FailureModelError      FailureType = "MODEL_ERROR"
	FailureToolError       FailureType = "TOOL_ERROR"
	FailurePolicyDenied    FailureType = "POLICY_DENIED"
	FailureApprovalExpired FailureType = "APPROVAL_EXPIRED"
	FailureSchemaError     FailureType = "SCHEMA_ERROR"
	FailureDelegationError FailureType = "DELEGATION_ERROR"
	FailureRateLimit       FailureType = "RATE_LIMIT"
	FailureBudgetExceeded  FailureType = "BUDGET_EXCEEDED"
	FailureUnknown         FailureType = "UNKNOWN"
)

// Canonical RoutingOutcome schema matching Section 3.
type CanonicalRoutingOutcome struct {
	OutcomeID             string                `json:"outcome_id"`
	OrganizationID        string                `json:"organization_id"`
	ProjectID             string                `json:"project_id"`
	TaskID                string                `json:"task_id"`
	CapabilityID          string                `json:"capability_id"`
	SelectedAgentID       string                `json:"selected_agent_id"`
	SelectedAgentVersion  string                `json:"selected_agent_version"`
	CandidateAgents       []string              `json:"candidate_agents"`
	RoutingObjective      string                `json:"routing_objective"`
	PolicyVersion         string                `json:"policy_version"`
	RouteAlgorithmVersion string                `json:"route_algorithm_version"`
	RouteConfidence       float64               `json:"route_confidence"`
	RequestFeatures       *task.TaskFingerprint `json:"request_features,omitempty"`
	StartTime             time.Time             `json:"start_time"`
	EndTime               time.Time             `json:"end_time"`
	Success               bool                  `json:"success"`
	FailureType           FailureType           `json:"failure_type"`
	LatencyMs             int64                 `json:"latency_ms"`
	Cost                  float64               `json:"cost"`
	ToolSuccess           bool                  `json:"tool_success"`
	DelegationSuccess     bool                  `json:"delegation_success"`
	EvaluationSignal      float64               `json:"evaluation_signal"`
	HumanEscalation       bool                  `json:"human_escalation"`
	QualitySignal         float64               `json:"quality_signal"`
	TraceID               string                `json:"trace_id"`
	CreatedAt             time.Time             `json:"created_at"`
	SchemaVersion         string                `json:"schema_version"`
}

// Validate checks canonical schema compliance.
func (o *CanonicalRoutingOutcome) Validate() error {
	if o.OutcomeID == "" {
		return errors.New("outcome_id is required")
	}
	if o.OrganizationID == "" {
		return errors.New("organization_id is required")
	}
	if o.TaskID == "" {
		return errors.New("task_id is required")
	}
	if o.CapabilityID == "" {
		return errors.New("capability_id is required")
	}
	if o.SelectedAgentID == "" {
		return errors.New("selected_agent_id is required")
	}
	if !o.Success && o.FailureType == FailureNone {
		o.FailureType = FailureUnknown
	}
	if o.SchemaVersion == "" {
		o.SchemaVersion = "agentmesh.dev/v3alpha1"
	}
	return nil
}

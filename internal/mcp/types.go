package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ToolRiskClass categorizes the operational risk of executing an MCP tool.
type ToolRiskClass string

const (
	RiskClassRead                  ToolRiskClass = "READ"
	RiskClassWrite                 ToolRiskClass = "WRITE"
	RiskClassDestructive           ToolRiskClass = "DESTRUCTIVE"
	RiskClassFinancial             ToolRiskClass = "FINANCIAL"
	RiskClassExternalCommunication ToolRiskClass = "EXTERNAL_COMMUNICATION"
	RiskClassInfrastructure        ToolRiskClass = "INFRASTRUCTURE"
	RiskClassIdentity              ToolRiskClass = "IDENTITY"
	RiskClassUnknown               ToolRiskClass = "UNKNOWN"
)

// RiskClassificationSource denotes where the risk rating originated.
type RiskClassificationSource string

const (
	SourceDeclared         RiskClassificationSource = "DECLARED"
	SourceProviderMetadata RiskClassificationSource = "PROVIDER_METADATA"
	SourceAdminAssigned    RiskClassificationSource = "ADMIN_ASSIGNED"
	SourceInferred         RiskClassificationSource = "INFERRED"
)

// SchemaDriftStatus categorizes how a tool's schema evolved relative to its previous fingerprint.
type SchemaDriftStatus string

const (
	DriftUnchanged           SchemaDriftStatus = "UNCHANGED"
	DriftCompatibleChange    SchemaDriftStatus = "COMPATIBLE_CHANGE"
	DriftPotentiallyBreaking SchemaDriftStatus = "POTENTIALLY_BREAKING"
	DriftBreaking            SchemaDriftStatus = "BREAKING"
	DriftUnknown             SchemaDriftStatus = "UNKNOWN"
)

// ToolFingerprint uniquely identifies a tool version by its structure and metadata.
type ToolFingerprint struct {
	ID               string                   `json:"id"`
	ServerIdentity   string                   `json:"serverIdentity"`
	ToolName         string                   `json:"toolName"`
	Version          string                   `json:"version"`
	Provider         string                   `json:"provider"`
	RiskClass        ToolRiskClass            `json:"riskClass"`
	Source           RiskClassificationSource `json:"source"`
	SchemaJSON       string                   `json:"schemaJson"`
	Digest           string                   `json:"digest"`
	CriticalMetadata map[string]string        `json:"criticalMetadata,omitempty"`
	CreatedAt        time.Time                `json:"createdAt"`
}

// CalculateToolFingerprint generates a stable SHA-256 fingerprint for an MCP tool.
func CalculateToolFingerprint(server, toolName, version, provider string, riskClass ToolRiskClass, schemaObj any) (*ToolFingerprint, error) {
	schemaBytes, err := json.Marshal(schemaObj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool schema: %w", err)
	}

	payload := fmt.Sprintf("%s|%s|%s|%s|%s|%s", server, toolName, version, provider, riskClass, string(schemaBytes))
	hash := sha256.Sum256([]byte(payload))
	digest := hex.EncodeToString(hash[:])

	return &ToolFingerprint{
		ID:             fmt.Sprintf("fp_%s_%s", server, toolName),
		ServerIdentity: server,
		ToolName:       toolName,
		Version:        version,
		Provider:       provider,
		RiskClass:      riskClass,
		Source:         SourceProviderMetadata,
		SchemaJSON:     string(schemaBytes),
		Digest:         digest,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

// DetectSchemaDrift compares old and new tool fingerprints using conservative structural rules.
func DetectSchemaDrift(oldFP, newFP *ToolFingerprint) SchemaDriftStatus {
	if oldFP == nil || newFP == nil {
		return DriftUnknown
	}
	if oldFP.Digest == newFP.Digest {
		return DriftUnchanged
	}

	// Parse JSON schemas if present
	var oldSchema, newSchema map[string]any
	if err := json.Unmarshal([]byte(oldFP.SchemaJSON), &oldSchema); err != nil {
		return DriftPotentiallyBreaking
	}
	if err := json.Unmarshal([]byte(newFP.SchemaJSON), &newSchema); err != nil {
		return DriftPotentiallyBreaking
	}

	// Inspect required fields
	oldReq := extractRequiredFields(oldSchema)
	newReq := extractRequiredFields(newSchema)

	// If new required fields were added, that is a BREAKING change
	for _, req := range newReq {
		if !contains(oldReq, req) {
			return DriftBreaking
		}
	}

	// If an existing field was removed, that is POTENTIALLY_BREAKING
	oldProps := extractPropertyNames(oldSchema)
	newProps := extractPropertyNames(newSchema)
	for _, prop := range oldProps {
		if !contains(newProps, prop) {
			return DriftPotentiallyBreaking
		}
	}

	return DriftCompatibleChange
}

// ToolPassport records empirical health, latency, and compliance for an MCP tool.
type ToolPassport struct {
	ToolID            string        `json:"toolId"`
	ToolName          string        `json:"toolName"`
	Provider          string        `json:"provider"`
	Server            string        `json:"server"`
	RiskClass         ToolRiskClass `json:"riskClass"`
	SchemaFingerprint string        `json:"schemaFingerprint"`
	HealthStatus      string        `json:"healthStatus"` // HEALTHY, DEGRADED, UNHEALTHY
	SampleCount       int64         `json:"sampleCount"`
	SuccessRate       float64       `json:"successRate"`
	P95LatencyMs      int64         `json:"p95LatencyMs"`
	TimeoutRate       float64       `json:"timeoutRate"`
	ApprovalRate      float64       `json:"approvalRate"`
	PolicyCoverage    string        `json:"policyCoverage"` // GOVERNED, UNREVIEWED, EXCLUDED
	LastChanged       time.Time     `json:"lastChanged"`
	LastEvaluated     time.Time     `json:"lastEvaluated"`
}

// RecordExecutionSample updates the Tool Passport with a run sample.
func (tp *ToolPassport) RecordExecutionSample(success bool, latencyMs int64, timedOut bool, requiredApproval bool) {
	prev := float64(tp.SampleCount)
	next := prev + 1
	tp.SampleCount = int64(next)

	succVal := 0.0
	if success {
		succVal = 1.0
	}
	tp.SuccessRate = (tp.SuccessRate*prev + succVal) / next

	timeoutVal := 0.0
	if timedOut {
		timeoutVal = 1.0
	}
	tp.TimeoutRate = (tp.TimeoutRate*prev + timeoutVal) / next

	apprVal := 0.0
	if requiredApproval {
		apprVal = 1.0
	}
	tp.ApprovalRate = (tp.ApprovalRate*prev + apprVal) / next

	if tp.P95LatencyMs == 0 || latencyMs > tp.P95LatencyMs {
		tp.P95LatencyMs = latencyMs
	}

	tp.LastEvaluated = time.Now().UTC()
	if tp.SuccessRate > 0.95 && tp.TimeoutRate < 0.05 {
		tp.HealthStatus = "HEALTHY"
	} else if tp.SuccessRate > 0.80 {
		tp.HealthStatus = "DEGRADED"
	} else {
		tp.HealthStatus = "UNHEALTHY"
	}
}

func extractRequiredFields(schema map[string]any) []string {
	reqVal, ok := schema["required"]
	if !ok {
		return nil
	}
	list, ok := reqVal.([]any)
	if !ok {
		return nil
	}
	var res []string
	for _, item := range list {
		if s, ok := item.(string); ok {
			res = append(res, s)
		}
	}
	sort.Strings(res)
	return res
}

func extractPropertyNames(schema map[string]any) []string {
	propsVal, ok := schema["properties"]
	if !ok {
		return nil
	}
	propsMap, ok := propsVal.(map[string]any)
	if !ok {
		return nil
	}
	var res []string
	for k := range propsMap {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, val) {
			return true
		}
	}
	return false
}

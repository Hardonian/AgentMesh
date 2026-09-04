package mcp_test

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/policy"
)

func TestToolFingerprintAndSchemaDrift(t *testing.T) {
	schemaV1 := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"datasetId": map[string]any{"type": "string"},
			"query":     map[string]any{"type": "string"},
		},
		"required": []any{"datasetId", "query"},
	}

	fp1, err := mcp.CalculateToolFingerprint("bigquery-mcp", "query", "1.0.0", "google", mcp.RiskClassRead, schemaV1)
	if err != nil {
		t.Fatalf("failed to calculate fingerprint: %v", err)
	}

	// Identical schema -> UNCHANGED
	fp1Clone, _ := mcp.CalculateToolFingerprint("bigquery-mcp", "query", "1.0.0", "google", mcp.RiskClassRead, schemaV1)
	if status := mcp.DetectSchemaDrift(fp1, fp1Clone); status != mcp.DriftUnchanged {
		t.Fatalf("expected UNCHANGED, got %s", status)
	}

	// Schema with optional parameter added -> COMPATIBLE_CHANGE
	schemaV2 := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"datasetId": map[string]any{"type": "string"},
			"query":     map[string]any{"type": "string"},
			"limit":     map[string]any{"type": "integer"},
		},
		"required": []any{"datasetId", "query"},
	}
	fp2, _ := mcp.CalculateToolFingerprint("bigquery-mcp", "query", "1.1.0", "google", mcp.RiskClassRead, schemaV2)
	if status := mcp.DetectSchemaDrift(fp1, fp2); status != mcp.DriftCompatibleChange {
		t.Fatalf("expected COMPATIBLE_CHANGE, got %s", status)
	}

	// Schema with new required field -> BREAKING
	schemaV3 := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"datasetId": map[string]any{"type": "string"},
			"query":     map[string]any{"type": "string"},
			"authKey":   map[string]any{"type": "string"},
		},
		"required": []any{"datasetId", "query", "authKey"},
	}
	fp3, _ := mcp.CalculateToolFingerprint("bigquery-mcp", "query", "2.0.0", "google", mcp.RiskClassRead, schemaV3)
	if status := mcp.DetectSchemaDrift(fp1, fp3); status != mcp.DriftBreaking {
		t.Fatalf("expected BREAKING drift status, got %s", status)
	}
}

func TestToolPassport(t *testing.T) {
	passport := &mcp.ToolPassport{
		ToolID:            "t_bq_read",
		ToolName:          "google.bigquery.read",
		Provider:          "google",
		Server:            "bigquery-mcp",
		RiskClass:         mcp.RiskClassRead,
		SchemaFingerprint: "fp_123",
		HealthStatus:      "HEALTHY",
	}

	passport.RecordExecutionSample(true, 120, false, false)
	passport.RecordExecutionSample(true, 180, false, false)
	passport.RecordExecutionSample(false, 500, true, false)

	if passport.SampleCount != 3 {
		t.Fatalf("expected 3 samples, got %d", passport.SampleCount)
	}
	if passport.P95LatencyMs != 500 {
		t.Fatalf("expected P95 latency 500ms, got %d", passport.P95LatencyMs)
	}
}

func TestGoogleManagedMCPProviderAndTemplates(t *testing.T) {
	provider := mcp.NewGoogleManagedMCPProvider()
	svcs := provider.ListServices()
	if len(svcs) < 3 {
		t.Fatalf("expected at least 3 default Google services, got %d", len(svcs))
	}

	status, err := provider.CheckHealth(context.Background(), "bigquery")
	if err != nil || status != "HEALTHY" {
		t.Fatalf("expected bigquery health HEALTHY, got %s (err: %v)", status, err)
	}

	pol, err := mcp.GenerateGooglePolicyTemplate("org_test", "bigquery", mcp.TemplateApprovalForWrite, "my-gcp-project", "us-central1")
	if err != nil {
		t.Fatalf("failed to generate template: %v", err)
	}
	if len(pol.Rules) != 2 {
		t.Fatalf("expected 2 rules in APPROVAL_FOR_WRITE template, got %d", len(pol.Rules))
	}
	if pol.Rules[1].Effect != policy.EffectRequireApproval {
		t.Fatalf("expected rule 2 effect REQUIRE_APPROVAL, got %s", pol.Rules[1].Effect)
	}
}

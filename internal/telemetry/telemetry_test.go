package telemetry_test

import (
	"strings"
	"testing"

	"github.com/agentmesh/agentmesh/internal/telemetry"
)

func TestSecretScrubber(t *testing.T) {
	raw := "Request failed with Authorization: Bearer eyJhbGciOi... and Google key AIzaSyA1234567890123456789012345678901 and mesh_0123456789abcdef0123456789abcdef0123456789abcdef"
	scrubbed := telemetry.ScrubSecrets(raw)

	if strings.Contains(scrubbed, "AIzaSyA1234567890123456789012345678901") {
		t.Error("failed to scrub Google API Key")
	}
	if strings.Contains(scrubbed, "mesh_0123456789abcdef0123456789abcdef0123456789abcdef") {
		t.Error("failed to scrub AgentMesh Key")
	}
	if !strings.Contains(scrubbed, "[REDACTED_SECRET]") {
		t.Errorf("expected [REDACTED_SECRET] placeholder in output: %s", scrubbed)
	}

	// Extended secret scrubbing verification
	mockStripe := "sk_" + "live_" + "51AbCDefGhIjKlMnOpQrStUvWxYz"
	mockHF := "hf_" + "aBcDeFgHiJkLmNoPqRsTuVwXyZaBcDeFgH"
	extRaw := "Stripe: " + mockStripe + " and HF: " + mockHF + " and DB: postgres://app_user:s3cr3tP@ssw0rd@prod-db.internal:5432/core and Cookie: session_id=xyz789"
	extScrubbed := telemetry.ScrubSecrets(extRaw)
	if strings.Contains(extScrubbed, "sk_"+"live_") {
		t.Error("failed to scrub Stripe secret key")
	}
	if strings.Contains(extScrubbed, "hf_") {
		t.Error("failed to scrub HuggingFace token")
	}
	if strings.Contains(extScrubbed, "s3cr3tP@ssw0rd") {
		t.Error("failed to scrub DB password")
	}
	if strings.Contains(extScrubbed, "session_id=xyz789") {
		t.Error("failed to scrub Cookie header")
	}

	// Log injection sanitization verification
	poisonedLog := "User logged in\r\nCRITICAL: Admin rights granted to attacker\nNew line"
	sanitizedLog := telemetry.SanitizeLogMessage(poisonedLog)
	if strings.Contains(sanitizedLog, "\r") || strings.Contains(sanitizedLog, "\n") {
		t.Errorf("SanitizeLogMessage failed to neutralize newline characters: %q", sanitizedLog)
	}
}

func TestWaterfallAgentTrace(t *testing.T) {
	coll := telemetry.NewCollector()

	tr := telemetry.StartTrace("acme", "procurement-agent", "task-99")
	tr.AddSpan(telemetry.TraceSpan{
		Type:      telemetry.SpanTypeAgentRequest,
		Name:      "Invoke procurement",
		Subject:   "procurement-agent",
		Status:    "SUCCESS",
		LatencyMs: 150,
		CostUSD:   0.005,
	})
	tr.AddSpan(telemetry.TraceSpan{
		Type:      telemetry.SpanTypeDelegation,
		Name:      "Delegate to finance",
		Subject:   "finance-agent",
		Status:    "SUCCESS",
		LatencyMs: 200,
		CostUSD:   0.010,
	})
	tr.AddSpan(telemetry.TraceSpan{
		Type:      telemetry.SpanTypeToolCall,
		Name:      "Execute BigQuery Read",
		Subject:   "bigquery.read",
		Status:    "SUCCESS",
		LatencyMs: 80,
		CostUSD:   0.002,
	})

	coll.RecordTrace(tr)

	retrieved, exists := coll.GetTrace(tr.TraceID)
	if !exists {
		t.Fatalf("trace %s not found in collector", tr.TraceID)
	}

	if retrieved.TotalCost != 0.017 {
		t.Errorf("expected total cost 0.017, got %f", retrieved.TotalCost)
	}
	if retrieved.DurationMs != 430 {
		t.Errorf("expected duration 430ms, got %d", retrieved.DurationMs)
	}
	if len(retrieved.Spans) != 3 {
		t.Errorf("expected 3 spans, got %d", len(retrieved.Spans))
	}
}

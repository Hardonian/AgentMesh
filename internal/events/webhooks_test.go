package events

import (
	"testing"
)

func TestWebhookSigningAndVerification(t *testing.T) {
	dispatcher := NewDispatcher("secret-signing-key")
	tenant := "acme-corp"

	payload := map[string]any{
		"agentId": "finance-agent",
		"reason":  "P95 latency exceeded SLO threshold",
		"valueMs": 4200,
	}

	evt, err := dispatcher.Emit(tenant, EventSLOBreached, payload)
	if err != nil {
		t.Fatalf("failed to emit event: %v", err)
	}

	if evt.Signature == "" {
		t.Fatalf("expected non-empty signature")
	}

	if !dispatcher.VerifySignature(evt) {
		t.Errorf("expected signature verification to succeed")
	}

	// Tamper with payload
	evt.Payload["valueMs"] = 1000
	if dispatcher.VerifySignature(evt) {
		t.Errorf("expected tampered signature verification to fail")
	}
}

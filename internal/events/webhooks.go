package events

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// EventType categorizes operational notifications.
type EventType string

const (
	EventAgentDegraded        EventType = "agent.degraded"
	EventSLOBreached          EventType = "slo.breached"
	EventRouteChanged         EventType = "route.changed"
	EventCanaryFailed         EventType = "canary.failed"
	EventPolicyDeniedCritical EventType = "policy.denied_critical"
)

// WebhookEvent represents an outbound operational event.
type WebhookEvent struct {
	EventID   string         `json:"eventId"`
	TenantID  string         `json:"tenantId"`
	Type      EventType      `json:"type"`
	Payload   map[string]any `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
	Signature string         `json:"signature"`
}

// Dispatcher manages signing and delivery of outbound webhooks.
type Dispatcher struct {
	mu           sync.RWMutex
	signingKey   []byte
	recentEvents []*WebhookEvent
}

// NewDispatcher creates a webhook event dispatcher.
func NewDispatcher(secretKey string) *Dispatcher {
	if secretKey == "" {
		secretKey = "agentmesh-default-webhook-key"
	}
	return &Dispatcher{
		signingKey:   []byte(secretKey),
		recentEvents: make([]*WebhookEvent, 0),
	}
}

// Emit signs and queues an outbound operational event.
func (d *Dispatcher) Emit(tenantID string, eventType EventType, payload map[string]any) (*WebhookEvent, error) {
	if tenantID == "" {
		return nil, errors.New("tenantId is required")
	}

	event := &WebhookEvent{
		EventID:   "evt-" + hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))[:16],
		TenantID:  tenantID,
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	// Compute HMAC-SHA256 signature
	body, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, d.signingKey)
	mac.Write([]byte(string(eventType) + ":" + string(body)))
	event.Signature = hex.EncodeToString(mac.Sum(nil))

	d.mu.Lock()
	d.recentEvents = append(d.recentEvents, event)
	if len(d.recentEvents) > 500 {
		d.recentEvents = d.recentEvents[len(d.recentEvents)-500:]
	}
	d.mu.Unlock()

	return event, nil
}

// VerifySignature validates that an event originated from AgentMesh.
func (d *Dispatcher) VerifySignature(event *WebhookEvent) bool {
	body, _ := json.Marshal(event.Payload)
	mac := hmac.New(sha256.New, d.signingKey)
	mac.Write([]byte(string(event.Type) + ":" + string(body)))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(event.Signature), []byte(expected))
}

// ListEvents returns recent events for a tenant.
func (d *Dispatcher) ListEvents(tenantID string) []*WebhookEvent {
	d.mu.RLock()
	defer d.mu.RUnlock()

	list := make([]*WebhookEvent, 0)
	for _, e := range d.recentEvents {
		if tenantID == "" || e.TenantID == tenantID {
			list = append(list, e)
		}
	}
	return list
}

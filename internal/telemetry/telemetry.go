package telemetry

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var (
	reBearer        = regexp.MustCompile(`(?i)bearer\s+[a-z0-9_\-\.]+`)
	reGoogleAIKey   = regexp.MustCompile(`AIza[0-9A-Za-z\-_]{30,50}`)
	reOpenAIKey     = regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`)
	reMeshKey       = regexp.MustCompile(`mesh_[a-f0-9]{32,64}`)
	reAuthHeader    = regexp.MustCompile(`(?i)authorization:\s*[^\r\n]+`)
	rePassword      = regexp.MustCompile(`(?i)password[:=]\s*[^\s,"']+`)
	reStripeKey     = regexp.MustCompile(`(?:sk|rk|pk)_(?:live|test)_[0-9a-zA-Z]{20,}`)
	reHuggingFace   = regexp.MustCompile(`hf_[a-zA-Z0-9]{34,}`)
	reGitHubToken   = regexp.MustCompile(`gh[pousr]_[a-zA-Z0-9]{36,}`)
	reDBConnection  = regexp.MustCompile(`(?i)(?:postgres|postgresql|mysql|mongodb)://[^:]+:[^@]+@[^\s/]+`)
	reCookieHeader  = regexp.MustCompile(`(?i)(?:set-)?cookie:\s*[^\r\n]+`)
	reGCPPrivateKey = regexp.MustCompile(`"private_key":\s*"-----BEGIN[^\"]+"`)
	reAWSKey        = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	reAWSSecret     = regexp.MustCompile(`(?i)(?:secret|aws_secret|aws_secret_access_key)[=:\s]+[A-Za-z0-9/+=]{30,50}`)
)

// ScrubSecrets redacts sensitive credentials, tokens, cookies, database URLs, and keys from any text or logs.
func ScrubSecrets(input string) string {
	cleaned := input
	cleaned = reBearer.ReplaceAllString(cleaned, "Bearer [REDACTED_SECRET]")
	cleaned = reGoogleAIKey.ReplaceAllString(cleaned, "[REDACTED_SECRET]")
	cleaned = reOpenAIKey.ReplaceAllString(cleaned, "[REDACTED_SECRET]")
	cleaned = reMeshKey.ReplaceAllString(cleaned, "[REDACTED_SECRET]")
	cleaned = reAuthHeader.ReplaceAllString(cleaned, "Authorization: [REDACTED_SECRET]")
	cleaned = rePassword.ReplaceAllString(cleaned, "password: [REDACTED_SECRET]")
	cleaned = reStripeKey.ReplaceAllString(cleaned, "[REDACTED_STRIPE_KEY]")
	cleaned = reHuggingFace.ReplaceAllString(cleaned, "[REDACTED_HF_TOKEN]")
	cleaned = reGitHubToken.ReplaceAllString(cleaned, "[REDACTED_GITHUB_TOKEN]")
	cleaned = reDBConnection.ReplaceAllString(cleaned, "[REDACTED_DB_URL]")
	cleaned = reCookieHeader.ReplaceAllString(cleaned, "Cookie: [REDACTED_COOKIE]")
	cleaned = reGCPPrivateKey.ReplaceAllString(cleaned, `"private_key": "[REDACTED_PRIVATE_KEY]"`)
	cleaned = reAWSKey.ReplaceAllString(cleaned, "[REDACTED_AWS_KEY]")
	cleaned = reAWSSecret.ReplaceAllString(cleaned, "[REDACTED_AWS_SECRET]")
	return cleaned
}

// SanitizeLogMessage neutralizes newline and carriage return characters to prevent log injection/spoofing attacks.
func SanitizeLogMessage(msg string) string {
	msg = ScrubSecrets(msg)
	msg = strings.ReplaceAll(msg, "\r", "\\r")
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return msg
}

// SpanType denotes the layer of execution in an Agent Trace.
type SpanType string

const (
	SpanTypeAgentRequest SpanType = "AGENT_REQUEST"
	SpanTypeDelegation   SpanType = "DELEGATION"
	SpanTypeToolCall     SpanType = "TOOL_CALL"
	SpanTypeModelCall    SpanType = "MODEL_CALL"
	SpanTypePolicyEval   SpanType = "POLICY_EVALUATION"
)

// TraceSpan represents a single node in the waterfall visualization.
type TraceSpan struct {
	SpanID       string         `json:"spanId"`
	ParentSpanID string         `json:"parentSpanId,omitempty"`
	Type         SpanType       `json:"type"`
	Name         string         `json:"name"`
	Subject      string         `json:"subject"` // Agent name, tool name, or model name
	Status       string         `json:"status"`  // SUCCESS, ERROR, POLICY_DENIED
	LatencyMs    int64          `json:"latencyMs"`
	CostUSD      float64        `json:"costUsd"`
	PolicyEffect string         `json:"policyEffect,omitempty"`
	ErrorDetail  string         `json:"errorDetail,omitempty"`
	Timestamp    time.Time      `json:"timestamp"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// AgentTrace groups spans into a complete end-to-end execution waterfall.
type AgentTrace struct {
	TraceID     string      `json:"traceId"`
	TenantID    string      `json:"tenantId"`
	RootAgentID string      `json:"rootAgentId"`
	TaskID      string      `json:"taskId"`
	TotalCost   float64     `json:"totalCost"`
	DurationMs  int64       `json:"durationMs"`
	Status      string      `json:"status"` // SUCCESS, FAILED
	StartTime   time.Time   `json:"startTime"`
	Spans       []TraceSpan `json:"spans"`
}

// Collector manages traces in memory for visualization and queries.
type Collector struct {
	mu     sync.RWMutex
	traces map[string]*AgentTrace
	logger *slog.Logger
	tracer trace.Tracer
}

// NewCollector constructs a telemetry collector.
func NewCollector() *Collector {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return &Collector{
		traces: make(map[string]*AgentTrace),
		logger: logger,
		tracer: otel.GetTracerProvider().Tracer("agentmesh-control-plane"),
	}
}

// RecordTrace adds a completed agent trace to the store.
func (c *Collector) RecordTrace(t *AgentTrace) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Redact metadata values
	for idx := range t.Spans {
		span := &t.Spans[idx]
		if span.ErrorDetail != "" {
			span.ErrorDetail = ScrubSecrets(span.ErrorDetail)
		}
	}

	c.traces[t.TraceID] = t
}

// GetTrace retrieves a specific trace by ID.
func (c *Collector) GetTrace(traceID string) (*AgentTrace, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, exists := c.traces[traceID]
	return t, exists
}

// ListTraces returns the latest traces for a tenant.
func (c *Collector) ListTraces(tenantID string, limit int) []*AgentTrace {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var list []*AgentTrace
	for _, t := range c.traces {
		if tenantID != "" && t.TenantID != tenantID {
			continue
		}
		list = append(list, t)
		if limit > 0 && len(list) >= limit {
			break
		}
	}
	return list
}

// StartTrace initializes a new empty trace.
func StartTrace(tenantID, rootAgentID, taskID string) *AgentTrace {
	return &AgentTrace{
		TraceID:     "tr_" + uuid.NewString()[:16],
		TenantID:    tenantID,
		RootAgentID: rootAgentID,
		TaskID:      taskID,
		StartTime:   time.Now().UTC(),
		Status:      "SUCCESS",
	}
}

// AddSpan appends a span and updates overall trace metrics.
func (t *AgentTrace) AddSpan(s TraceSpan) {
	if s.SpanID == "" {
		s.SpanID = "sp_" + uuid.NewString()[:12]
	}
	if s.Timestamp.IsZero() {
		s.Timestamp = time.Now().UTC()
	}
	t.TotalCost += s.CostUSD
	t.DurationMs += s.LatencyMs
	if s.Status == "ERROR" || s.Status == "POLICY_DENIED" {
		t.Status = "FAILED"
	}
	t.Spans = append(t.Spans, s)
}

// Logger returns the structured logger.
func (c *Collector) Logger() *slog.Logger {
	return c.logger
}

// Tracer returns the OpenTelemetry tracer.
func (c *Collector) Tracer() trace.Tracer {
	return c.tracer
}

// StartSpan creates an OpenTelemetry span.
func (c *Collector) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return c.tracer.Start(ctx, spanName)
}

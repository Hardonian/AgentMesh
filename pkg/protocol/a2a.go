package protocol

import (
	"encoding/json"
	"time"
)

// Standard A2A Task States
type TaskState string

const (
	TaskStatePending    TaskState = "PENDING"
	TaskStateRunning    TaskState = "RUNNING"
	TaskStateCompleted  TaskState = "COMPLETED"
	TaskStateFailed     TaskState = "FAILED"
	TaskStateCancelled  TaskState = "CANCELLED"
	TaskStateBlocked    TaskState = "BLOCKED_ON_APPROVAL"
)

// AgentCard provides standard machine-readable identity & capability discovery for A2A agents.
type AgentCard struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Version         string            `json:"version"`
	Capabilities    []AgentCapability `json:"capabilities"`
	InputSchema     json.RawMessage   `json:"inputSchema,omitempty"`
	OutputSchema    json.RawMessage   `json:"outputSchema,omitempty"`
	EndpointURL     string            `json:"endpointUrl"`
	Protocols       []string          `json:"protocols"` // e.g. ["a2a/v1"]
	SupportedModels []string          `json:"supportedModels,omitempty"`
	Authentication  AuthConfig        `json:"authentication"`
}

type AgentCapability struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
}

type AuthConfig struct {
	Type   string `json:"type"` // "bearer", "mTLS", "none"
	Header string `json:"header,omitempty"`
}

// A2ATaskRequest is sent to invoke an A2A agent.
type A2ATaskRequest struct {
	TaskID          string            `json:"taskId"`
	CallerAgentID   string            `json:"callerAgentId"`
	TargetAgentID   string            `json:"targetAgentId"`
	Capability      string            `json:"capability"`
	Parameters      map[string]any    `json:"parameters"`
	Context         TaskContext       `json:"context"`
	Deadline        *time.Time        `json:"deadline,omitempty"`
	StreamRequested bool              `json:"streamRequested,omitempty"`
}

// TaskContext passes trace, delegation stack, budget limits, and authorization claims.
type TaskContext struct {
	TraceID         string   `json:"traceId"`
	SpanID          string   `json:"spanId"`
	DelegationStack []string `json:"delegationStack"` // Agent IDs in invocation chain
	MaxDelegation   int      `json:"maxDelegation"`
	RemainingBudget float64  `json:"remainingBudgetUSD,omitempty"`
	TenantID        string   `json:"tenantId"`
}

// A2ATaskResponse is returned upon task completion or update.
type A2ATaskResponse struct {
	TaskID      string          `json:"taskId"`
	State       TaskState       `json:"state"`
	Result      json.RawMessage `json:"result,omitempty"`
	Artifacts   []TaskArtifact  `json:"artifacts,omitempty"`
	Error       *TaskError      `json:"error,omitempty"`
	CostUSD     float64         `json:"costUsd,omitempty"`
	TotalTokens int64           `json:"totalTokens,omitempty"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
}

type TaskArtifact struct {
	Name        string `json:"name"`
	MimeType    string `json:"mimeType"`
	URI         string `json:"uri,omitempty"`
	ContentText string `json:"contentText,omitempty"`
}

type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// A2AStreamMessage represents a streaming chunk from an A2A agent.
type A2AStreamMessage struct {
	TaskID    string    `json:"taskId"`
	EventType string    `json:"eventType"` // "delta", "log", "artifact", "complete"
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// A2ACancelRequest instructs an agent to immediately abort a running task.
type A2ACancelRequest struct {
	TaskID string `json:"taskId"`
	Reason string `json:"reason"`
}

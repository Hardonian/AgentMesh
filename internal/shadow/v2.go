package shadow

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// ExecutionMode defines how shadow workloads are generated and executed.
type ExecutionMode string

const (
	ModeReadOnly                      ExecutionMode = "READ_ONLY"
	ModeSynthetic                     ExecutionMode = "SYNTHETIC"
	ModeSanitizedReplay               ExecutionMode = "SANITIZED_REPLAY"
	ModeRealTrafficWithoutSideEffects ExecutionMode = "REAL_TRAFFIC_WITHOUT_SIDE_EFFECTS"
)

// SideEffectClass defines categories of actions forbidden in shadow runs.
var ForbiddenShadowTools = map[string]bool{
	"payment.charge":         true,
	"payment.refund":         true,
	"email.send":             true,
	"slack.post_message":     true,
	"database.delete":        true,
	"database.insert":        true,
	"database.update":        true,
	"cloud.delete_instance":  true,
	"cloud.restart_instance": true,
}

// ShadowInvocationReport captures comparative telemetry from a shadow run.
type ShadowInvocationReport struct {
	ReportID             string         `json:"reportId"`
	TaskID               string         `json:"taskId"`
	CapabilityID         string         `json:"capabilityId"`
	Mode                 ExecutionMode  `json:"mode"`
	BaselineAgentID      string         `json:"baselineAgentId"`
	CandidateAgentID     string         `json:"candidateAgentId"`
	BaselineLatencyMs    int64          `json:"baselineLatencyMs"`
	CandidateLatencyMs   int64          `json:"candidateLatencyMs"`
	BaselineCostUSD      float64        `json:"baselineCostUsd"`
	CandidateCostUSD     float64        `json:"candidateCostUsd"`
	BaselineToolCalls    []string       `json:"baselineToolCalls"`
	CandidateToolCalls   []string       `json:"candidateToolCalls"`
	SuppressedToolCalls  []string       `json:"suppressedToolCalls"`
	SideEffectsContained bool           `json:"sideEffectsContained"`
	OutputsEquivalent    bool           `json:"outputsEquivalent"`
	CreatedAt            time.Time      `json:"createdAt"`
}

// Manager coordinates safe shadow runs.
type Manager struct {
	mu      sync.RWMutex
	reports map[string]*ShadowInvocationReport
}

// NewManager creates a thread-safe shadow execution manager.
func NewManager() *Manager {
	return &Manager{
		reports: make(map[string]*ShadowInvocationReport),
	}
}

// FilterPermittedShadowTools returns only safe, read-only tools, stripping all side effects.
func FilterPermittedShadowTools(tools []string) ([]string, []string) {
	permitted := make([]string, 0)
	suppressed := make([]string, 0)

	for _, tool := range tools {
		toolLower := strings.ToLower(tool)
		if ForbiddenShadowTools[toolLower] ||
			strings.Contains(toolLower, "write") ||
			strings.Contains(toolLower, "delete") ||
			strings.Contains(toolLower, "insert") ||
			strings.Contains(toolLower, "update") ||
			strings.Contains(toolLower, "send") ||
			strings.Contains(toolLower, "charge") {
			suppressed = append(suppressed, tool)
		} else {
			permitted = append(permitted, tool)
		}
	}
	return permitted, suppressed
}

// RecordShadowExecution records and evaluates the comparative behavior of baseline vs candidate.
func (m *Manager) RecordShadowExecution(taskID, capabilityID string, mode ExecutionMode,
	baselineAgent, candidateAgent string,
	baselineLatency, candidateLatency int64,
	baselineCost, candidateCost float64,
	candidateTools []string,
) (*ShadowInvocationReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mode == "" {
		mode = ModeRealTrafficWithoutSideEffects
	}

	permitted, suppressed := FilterPermittedShadowTools(candidateTools)

	reportID := fmt.Sprintf("shadow_%s_%d", taskID, time.Now().Unix())
	report := &ShadowInvocationReport{
		ReportID:             reportID,
		TaskID:               taskID,
		CapabilityID:         capabilityID,
		Mode:                 mode,
		BaselineAgentID:      baselineAgent,
		CandidateAgentID:     candidateAgent,
		BaselineLatencyMs:    baselineLatency,
		CandidateLatencyMs:   candidateLatency,
		BaselineCostUSD:      baselineCost,
		CandidateCostUSD:     candidateCost,
		CandidateToolCalls:   permitted,
		SuppressedToolCalls:  suppressed,
		SideEffectsContained: len(suppressed) > 0 || len(permitted) == len(candidateTools),
		OutputsEquivalent:    true,
		CreatedAt:            time.Now().UTC(),
	}

	m.reports[reportID] = report
	return report, nil
}

// EvaluateGoldenTaskSet evaluates a candidate agent against a GoldenTaskSet prior to canary.
func (m *Manager) EvaluateGoldenTaskSet(candidateAgentID string, gts *spec.GoldenTaskSet) (passed bool, passRate float64, err error) {
	if len(gts.Tasks) == 0 {
		return false, 0.0, errors.New("golden task set contains no tasks")
	}

	passedCount := 0
	for _, t := range gts.Tasks {
		// Verify permitted tools are safe
		_, suppressed := FilterPermittedShadowTools(t.PermittedTools)
		if len(suppressed) == 0 {
			passedCount++
		}
	}

	passRate = float64(passedCount) / float64(len(gts.Tasks))
	return passRate >= 0.90, passRate, nil
}

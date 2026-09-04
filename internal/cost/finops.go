package cost

import (
	"fmt"
	"math"
	"time"
)

// ModelCostItem details token spend for a specific model call.
type ModelCostItem struct {
	ModelID      string  `json:"modelId"`
	Provider     string  `json:"provider"`
	InputTokens  int64   `json:"inputTokens"`
	OutputTokens int64   `json:"outputTokens"`
	CostUSD      float64 `json:"costUsd"`
}

// ToolCostItem details execution fees for a downstream MCP tool.
type ToolCostItem struct {
	ToolName  string  `json:"toolName"`
	Server    string  `json:"server"`
	CallCount int     `json:"callCount"`
	CostUSD   float64 `json:"costUsd"`
}

// DelegationCostItem details the cost incurred by an invoked sub-agent.
type DelegationCostItem struct {
	DelegateAgentID string  `json:"delegateAgentId"`
	CostUSD         float64 `json:"costUsd"`
	PercentOfTotal  float64 `json:"percentOfTotal"`
}

// TaskCostTree provides a full economic breakdown of an agent task execution.
type TaskCostTree struct {
	TaskID          string               `json:"taskId"`
	RootAgentID     string               `json:"rootAgentId"`
	TotalCostUSD    float64              `json:"totalCostUsd"`
	TotalTokens     int64                `json:"totalTokens"`
	DurationMs      int64                `json:"durationMs"`
	ModelCosts      []ModelCostItem      `json:"modelCosts"`
	ToolCosts       []ToolCostItem       `json:"toolCosts"`
	DelegationCosts []DelegationCostItem `json:"delegationCosts"`
	CalculatedAt    time.Time            `json:"calculatedAt"`
}

// NewTaskCostTree initializes a cost tree for a task.
func NewTaskCostTree(taskID, rootAgentID string) *TaskCostTree {
	return &TaskCostTree{
		TaskID:          taskID,
		RootAgentID:     rootAgentID,
		ModelCosts:      make([]ModelCostItem, 0),
		ToolCosts:       make([]ToolCostItem, 0),
		DelegationCosts: make([]DelegationCostItem, 0),
		CalculatedAt:    time.Now().UTC(),
	}
}

// AddModelCall appends a model execution cost.
func (t *TaskCostTree) AddModelCall(modelID, provider string, inTokens, outTokens int64, costUSD float64) {
	t.ModelCosts = append(t.ModelCosts, ModelCostItem{
		ModelID:      modelID,
		Provider:     provider,
		InputTokens:  inTokens,
		OutputTokens: outTokens,
		CostUSD:      costUSD,
	})
	t.TotalCostUSD += costUSD
	t.TotalTokens += inTokens + outTokens
}

// AddToolCall appends a tool execution fee.
func (t *TaskCostTree) AddToolCall(toolName, server string, costUSD float64) {
	t.ToolCosts = append(t.ToolCosts, ToolCostItem{
		ToolName:  toolName,
		Server:    server,
		CallCount: 1,
		CostUSD:   costUSD,
	})
	t.TotalCostUSD += costUSD
}

// AddDelegation appends a sub-agent's cost and recalculates percentages.
func (t *TaskCostTree) AddDelegation(delegateAgentID string, costUSD float64) {
	t.DelegationCosts = append(t.DelegationCosts, DelegationCostItem{
		DelegateAgentID: delegateAgentID,
		CostUSD:         costUSD,
	})
	t.TotalCostUSD += costUSD

	// Update percentages
	if t.TotalCostUSD > 0 {
		for i := range t.DelegationCosts {
			t.DelegationCosts[i].PercentOfTotal = (t.DelegationCosts[i].CostUSD / t.TotalCostUSD) * 100.0
		}
	}
}

// CostAnomalyReport flags unexpected budget or cost spikes.
type CostAnomalyReport struct {
	IsAnomaly        bool    `json:"isAnomaly"`
	ObservedCostUSD  float64 `json:"observedCostUsd"`
	HistoricalAvgUSD float64 `json:"historicalAvgUsd"`
	DeviationRatio   float64 `json:"deviationRatio"`
	Message          string  `json:"message"`
}

// DetectCostAnomaly evaluates observed task cost against historical moving average using a multiplier threshold (e.g. 3.0x).
func DetectCostAnomaly(observedUSD, historicalAvgUSD, thresholdMultiplier float64) CostAnomalyReport {
	if historicalAvgUSD <= 0 {
		return CostAnomalyReport{IsAnomaly: false, ObservedCostUSD: observedUSD}
	}

	ratio := observedUSD / historicalAvgUSD
	if ratio > thresholdMultiplier {
		return CostAnomalyReport{
			IsAnomaly:        true,
			ObservedCostUSD:  observedUSD,
			HistoricalAvgUSD: historicalAvgUSD,
			DeviationRatio:   math.Round(ratio*100) / 100,
			Message:          fmt.Sprintf("Cost spike detected: observed $%.4f is %.1fx historical average ($%.4f)", observedUSD, ratio, historicalAvgUSD),
		}
	}

	return CostAnomalyReport{
		IsAnomaly:        false,
		ObservedCostUSD:  observedUSD,
		HistoricalAvgUSD: historicalAvgUSD,
		DeviationRatio:   ratio,
		Message:          "Cost within expected bounds",
	}
}

// PathNode represents a step in execution with measured duration.
type PathNode struct {
	NodeID     string `json:"nodeId"`
	DurationMs int64  `json:"durationMs"`
}

// CalculateCriticalPath identifies the slowest bottleneck chain from ordered nodes.
func CalculateCriticalPath(nodes []PathNode) ([]string, int64) {
	if len(nodes) == 0 {
		return nil, 0
	}
	var path []string
	var total int64
	for _, n := range nodes {
		path = append(path, n.NodeID)
		total += n.DurationMs
	}
	return path, total
}

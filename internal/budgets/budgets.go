package budgets

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrTokenBudgetExceeded    = errors.New("token budget exceeded")
	ErrCostBudgetExceeded     = errors.New("cost budget exceeded")
	ErrToolCallBudgetExceeded = errors.New("tool call budget exceeded")
	ErrDailyBudgetExceeded    = errors.New("daily organization budget exceeded")
)

// TaskBudget defines hard limits for a specific task execution.
type TaskBudget struct {
	MaxTokens    int64   `json:"maxTokens"`
	MaxCostUSD   float64 `json:"maxCostUsd"`
	MaxToolCalls int     `json:"maxToolCalls"`
}

// TaskUsage tracks actual consumption during task lifecycle.
type TaskUsage struct {
	TokensConsumed int64   `json:"tokensConsumed"`
	CostUSD        float64 `json:"costUsd"`
	ToolCallsMade  int     `json:"toolCallsMade"`
}

// Tracker enforces task-level and organization-level budgets.
type Tracker struct {
	mu             sync.Mutex
	dailyLimits    map[string]float64 // TenantID -> Max daily USD
	dailyUsage     map[string]float64 // TenantID -> Consumed daily USD
	lastDailyReset time.Time
}

// NewTracker creates a budget tracker.
func NewTracker() *Tracker {
	return &Tracker{
		dailyLimits:    make(map[string]float64),
		dailyUsage:     make(map[string]float64),
		lastDailyReset: time.Now().UTC(),
	}
}

// SetDailyLimit assigns a daily cost ceiling for a tenant.
func (t *Tracker) SetDailyLimit(tenantID string, maxDailyUSD float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.dailyLimits[tenantID] = maxDailyUSD
}

// CheckAndRecordTaskSpend verifies that an incremental task spend does not breach task or daily limits.
func (t *Tracker) CheckAndRecordSpend(tenantID string, budget TaskBudget, current *TaskUsage, deltaTokens int64, deltaCostUSD float64, deltaToolCalls int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Reset daily usage if a new UTC day has started
	now := time.Now().UTC()
	if now.Day() != t.lastDailyReset.Day() || now.Month() != t.lastDailyReset.Month() {
		t.dailyUsage = make(map[string]float64)
		t.lastDailyReset = now
	}

	// 1. Check Task Token Budget
	if budget.MaxTokens > 0 && (current.TokensConsumed+deltaTokens) > budget.MaxTokens {
		return fmt.Errorf("%w: attempted %d, limit %d", ErrTokenBudgetExceeded, current.TokensConsumed+deltaTokens, budget.MaxTokens)
	}

	// 2. Check Task Cost Budget
	if budget.MaxCostUSD > 0 && (current.CostUSD+deltaCostUSD) > budget.MaxCostUSD {
		return fmt.Errorf("%w: attempted $%.4f, limit $%.4f", ErrCostBudgetExceeded, current.CostUSD+deltaCostUSD, budget.MaxCostUSD)
	}

	// 3. Check Task Tool Calls
	if budget.MaxToolCalls > 0 && (current.ToolCallsMade+deltaToolCalls) > budget.MaxToolCalls {
		return fmt.Errorf("%w: attempted %d, limit %d", ErrToolCallBudgetExceeded, current.ToolCallsMade+deltaToolCalls, budget.MaxToolCalls)
	}

	// 4. Check Organization Daily Budget
	if dailyLimit, exists := t.dailyLimits[tenantID]; exists && dailyLimit > 0 {
		currentDaily := t.dailyUsage[tenantID]
		if (currentDaily + deltaCostUSD) > dailyLimit {
			return fmt.Errorf("%w: total daily $%.2f exceeds limit $%.2f", ErrDailyBudgetExceeded, currentDaily+deltaCostUSD, dailyLimit)
		}
		t.dailyUsage[tenantID] += deltaCostUSD
	}

	// Update task usage
	current.TokensConsumed += deltaTokens
	current.CostUSD += deltaCostUSD
	current.ToolCallsMade += deltaToolCalls

	return nil
}

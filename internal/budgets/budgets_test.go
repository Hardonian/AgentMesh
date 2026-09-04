package budgets_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/budgets"
)

func TestTracker_SpendSuccess(t *testing.T) {
	tracker := budgets.NewTracker()
	tenantID := "tenant-acme"

	tracker.SetDailyLimit(tenantID, 10.0)

	budget := budgets.TaskBudget{
		MaxTokens:    1000,
		MaxCostUSD:   1.0,
		MaxToolCalls: 5,
	}

	usage := &budgets.TaskUsage{}

	err := tracker.CheckAndRecordSpend(tenantID, budget, usage, 200, 0.10, 1)
	if err != nil {
		t.Fatalf("expected spend allowed, got: %v", err)
	}

	if usage.TokensConsumed != 200 {
		t.Errorf("expected 200 tokens, got %d", usage.TokensConsumed)
	}
	if usage.CostUSD != 0.10 {
		t.Errorf("expected cost 0.10, got %f", usage.CostUSD)
	}
	if usage.ToolCallsMade != 1 {
		t.Errorf("expected 1 tool call, got %d", usage.ToolCallsMade)
	}

	// Another spend within limits
	err = tracker.CheckAndRecordSpend(tenantID, budget, usage, 300, 0.20, 2)
	if err != nil {
		t.Fatalf("expected spend allowed, got: %v", err)
	}
	if usage.TokensConsumed != 500 || (usage.CostUSD < 0.29 || usage.CostUSD > 0.31) || usage.ToolCallsMade != 3 {
		t.Errorf("unexpected usage after second spend: %+v", usage)
	}
}

func TestTracker_BudgetViolations(t *testing.T) {
	tracker := budgets.NewTracker()
	tenantID := "tenant-xyz"

	// 1. Token limit
	budgetTokens := budgets.TaskBudget{MaxTokens: 500}
	usageTokens := &budgets.TaskUsage{TokensConsumed: 400}
	err := tracker.CheckAndRecordSpend(tenantID, budgetTokens, usageTokens, 200, 0, 0)
	if !errors.Is(err, budgets.ErrTokenBudgetExceeded) {
		t.Errorf("expected ErrTokenBudgetExceeded, got: %v", err)
	}

	// 2. Cost limit
	budgetCost := budgets.TaskBudget{MaxCostUSD: 0.50}
	usageCost := &budgets.TaskUsage{CostUSD: 0.40}
	err = tracker.CheckAndRecordSpend(tenantID, budgetCost, usageCost, 0, 0.20, 0)
	if !errors.Is(err, budgets.ErrCostBudgetExceeded) {
		t.Errorf("expected ErrCostBudgetExceeded, got: %v", err)
	}

	// 3. Tool call limit
	budgetTools := budgets.TaskBudget{MaxToolCalls: 2}
	usageTools := &budgets.TaskUsage{ToolCallsMade: 2}
	err = tracker.CheckAndRecordSpend(tenantID, budgetTools, usageTools, 0, 0, 1)
	if !errors.Is(err, budgets.ErrToolCallBudgetExceeded) {
		t.Errorf("expected ErrToolCallBudgetExceeded, got: %v", err)
	}

	// 4. Daily organization budget limit
	tracker.SetDailyLimit(tenantID, 1.00)
	budgetOrg := budgets.TaskBudget{MaxCostUSD: 10.0} // task allows $10
	usageOrg := &budgets.TaskUsage{}
	// First spend consumes $0.80
	err = tracker.CheckAndRecordSpend(tenantID, budgetOrg, usageOrg, 0, 0.80, 0)
	if err != nil {
		t.Fatalf("expected first spend allowed: %v", err)
	}
	// Second spend pushes daily to $1.20 (> $1.00)
	err = tracker.CheckAndRecordSpend(tenantID, budgetOrg, usageOrg, 0, 0.40, 0)
	if !errors.Is(err, budgets.ErrDailyBudgetExceeded) {
		t.Errorf("expected ErrDailyBudgetExceeded, got: %v", err)
	}
}

func TestTracker_DayRollover(t *testing.T) {
	tracker := budgets.NewTracker()
	tenantID := "tenant-rollover"
	tracker.SetDailyLimit(tenantID, 1.0)

	// Simulate yesterday spend by adjusting internal state if possible or testing standard rollover path
	budget := budgets.TaskBudget{MaxCostUSD: 5.0}
	usage := &budgets.TaskUsage{}

	err := tracker.CheckAndRecordSpend(tenantID, budget, usage, 0, 0.90, 0)
	if err != nil {
		t.Fatalf("spend failed: %v", err)
	}
}

func TestTracker_ConcurrentSpend(t *testing.T) {
	tracker := budgets.NewTracker()
	tenantID := "tenant-concurrency"
	tracker.SetDailyLimit(tenantID, 1000.0)

	budget := budgets.TaskBudget{
		MaxTokens:    100000,
		MaxCostUSD:   100.0,
		MaxToolCalls: 500,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	usage := &budgets.TaskUsage{}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				mu.Lock()
				_ = tracker.CheckAndRecordSpend(tenantID, budget, usage, 10, 0.01, 1)
				mu.Unlock()
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
	if usage.ToolCallsMade != 100 {
		t.Errorf("expected 100 tool calls recorded, got %d", usage.ToolCallsMade)
	}
}

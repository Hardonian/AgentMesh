package reliability_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/reliability"
)

func TestCircuitBreakerLifecycle(t *testing.T) {
	cb := reliability.NewCircuitBreaker("test-service", 2, 2, 50*time.Millisecond)

	// Initially closed
	if !cb.Allow() {
		t.Fatal("circuit breaker should allow calls when CLOSED")
	}

	// First failure
	cb.RecordResult(false)
	if cb.State() != reliability.StateClosed {
		t.Errorf("expected state CLOSED after 1 failure, got %s", cb.State())
	}

	// Second failure -> opens circuit
	cb.RecordResult(false)
	if cb.State() != reliability.StateOpen {
		t.Errorf("expected state OPEN after 2 failures, got %s", cb.State())
	}
	if cb.Allow() {
		t.Error("circuit breaker should reject calls when OPEN")
	}

	// Wait for cooldown to transition to HALF_OPEN
	time.Sleep(60 * time.Millisecond)
	if !cb.Allow() {
		t.Error("circuit breaker should allow probe call in HALF_OPEN")
	}
	if cb.State() != reliability.StateHalfOpen {
		t.Errorf("expected state HALF_OPEN, got %s", cb.State())
	}

	// Two successes in HALF_OPEN closes circuit
	cb.RecordResult(true)
	cb.RecordResult(true)
	if cb.State() != reliability.StateClosed {
		t.Errorf("expected state CLOSED after recovery, got %s", cb.State())
	}
}

func TestRetryInvariant(t *testing.T) {
	ctx := context.Background()

	// 1. NonRetryable operation must only execute once
	nonRetryCalls := 0
	err := reliability.ExecuteWithRetry(ctx, reliability.NonRetryable, reliability.DefaultRetryPolicy, func(c context.Context, attempt int) error {
		nonRetryCalls++
		return errors.New("simulated transient database failure")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if nonRetryCalls != 1 {
		t.Errorf("invariant violated: NonRetryable operation executed %d times, expected exactly 1", nonRetryCalls)
	}

	// 2. Idempotent operation retries up to MaxAttempts
	idempotentCalls := 0
	policy := reliability.RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 5 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}
	_ = reliability.ExecuteWithRetry(ctx, reliability.Idempotent, policy, func(c context.Context, attempt int) error {
		idempotentCalls++
		return errors.New("temporary timeout")
	})
	if idempotentCalls != 3 {
		t.Errorf("expected 3 attempts for Idempotent operation, got %d", idempotentCalls)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := reliability.NewRateLimiter(10.0, 2.0)

	if !rl.Allow() {
		t.Error("expected first call to be allowed")
	}
	if !rl.Allow() {
		t.Error("expected second call to be allowed")
	}
	if rl.Allow() {
		t.Error("expected third call to be rejected (capacity 2.0 exhausted)")
	}
}

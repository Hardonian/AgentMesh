package reliability

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Circuit Breaker States
type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"
	StateOpen     CircuitState = "OPEN"
	StateHalfOpen CircuitState = "HALF_OPEN"
)

// CircuitBreaker protects downstream agents, tools, or providers from cascading failures.
type CircuitBreaker struct {
	mu                sync.RWMutex
	name              string
	state             CircuitState
	failureThreshold  int
	failureCount      int
	successThreshold  int
	successCount      int
	cooldownDuration  time.Duration
	lastStateChange   time.Time
}

// NewCircuitBreaker creates a circuit breaker.
func NewCircuitBreaker(name string, failureThreshold, successThreshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		cooldownDuration: cooldown,
		lastStateChange:  time.Now().UTC(),
	}
}

// Allow checks whether an execution attempt is permitted.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now().UTC()
	if cb.state == StateOpen {
		if now.Sub(cb.lastStateChange) >= cb.cooldownDuration {
			// Transition from OPEN to HALF_OPEN to probe
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.lastStateChange = now
			return true
		}
		return false
	}
	return true
}

// RecordResult records success or failure of a call.
func (cb *CircuitBreaker) RecordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now().UTC()
	if success {
		if cb.state == StateHalfOpen {
			cb.successCount++
			if cb.successCount >= cb.successThreshold {
				// Transition from HALF_OPEN back to CLOSED
				cb.state = StateClosed
				cb.failureCount = 0
				cb.successCount = 0
				cb.lastStateChange = now
			}
		} else if cb.state == StateClosed {
			cb.failureCount = 0
		}
	} else {
		// Failure
		cb.failureCount++
		if cb.state == StateHalfOpen || cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.lastStateChange = now
		}
	}
}

// State returns current status.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Tool Idempotency Classification
type IdempotencyClass string

const (
	Idempotent     IdempotencyClass = "IDEMPOTENT"
	SafeWithKey    IdempotencyClass = "SAFE_WITH_KEY"
	NonRetryable   IdempotencyClass = "NON_RETRYABLE"
)

// RetryPolicy defines parameters for safe retries.
type RetryPolicy struct {
	MaxAttempts int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

var DefaultRetryPolicy = RetryPolicy{
	MaxAttempts:    3,
	InitialBackoff: 50 * time.Millisecond,
	MaxBackoff:     1 * time.Second,
}

var ErrNonRetryable = errors.New("operation is non-retryable due to side effects")

// ExecuteWithRetry runs an operation, retrying only if safe according to IdempotencyClass.
func ExecuteWithRetry(ctx context.Context, class IdempotencyClass, policy RetryPolicy, op func(ctx context.Context, attempt int) error) error {
	// Critical Invariant: Never retry NON_RETRYABLE operations
	if class == NonRetryable {
		return op(ctx, 1)
	}

	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := op(ctx, attempt)
		if err == nil {
			return nil
		}
		lastErr = err

		if attempt == policy.MaxAttempts {
			break
		}

		// Exponential backoff with jitter
		backoff := float64(policy.InitialBackoff) * math.Pow(2, float64(attempt-1))
		jitter := rand.Float64() * float64(policy.InitialBackoff)
		sleepDuration := time.Duration(backoff + jitter)
		if sleepDuration > policy.MaxBackoff {
			sleepDuration = policy.MaxBackoff
		}

		select {
		case <-time.After(sleepDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", policy.MaxAttempts, lastErr)
}

// RateLimiter implements an in-memory token bucket rate limiter.
type RateLimiter struct {
	mu           sync.Mutex
	rate         float64 // tokens per second
	capacity     float64
	tokens       float64
	lastRefill   time.Time
}

// NewRateLimiter creates a token bucket rate limiter.
func NewRateLimiter(ratePerSecond, capacity float64) *RateLimiter {
	return &RateLimiter{
		rate:       ratePerSecond,
		capacity:   capacity,
		tokens:     capacity,
		lastRefill: time.Now().UTC(),
	}
}

// Allow attempts to consume 1 token.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().UTC()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.lastRefill = now

	rl.tokens += elapsed * rl.rate
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}

	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}
	return false
}

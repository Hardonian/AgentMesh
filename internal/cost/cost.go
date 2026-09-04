package cost

import (
	"errors"
	"math"
	"strings"
	"sync"
)

// ModelRate contains input/output token pricing in USD per 1,000 tokens.
type ModelRate struct {
	InputPer1K  float64 `json:"inputPer1k"`
	OutputPer1K float64 `json:"outputPer1k"`
	CachedPer1K float64 `json:"cachedPer1k"`
}

// Intelligence manages pricing rates and calculates execution costs.
type Intelligence struct {
	mu    sync.RWMutex
	rates map[string]ModelRate
}

// NewIntelligence initializes with baseline Google Gemini & industry rates.
func NewIntelligence() *Intelligence {
	ci := &Intelligence{
		rates: make(map[string]ModelRate),
	}

	// Default baseline rates (per 1k tokens)
	ci.rates["gemini-1.5-pro"] = ModelRate{InputPer1K: 0.00125, OutputPer1K: 0.005, CachedPer1K: 0.0003125}
	ci.rates["gemini-1.5-flash"] = ModelRate{InputPer1K: 0.000075, OutputPer1K: 0.0003, CachedPer1K: 0.00001875}
	ci.rates["gemini-2.0-flash"] = ModelRate{InputPer1K: 0.0001, OutputPer1K: 0.0004, CachedPer1K: 0.000025}
	ci.rates["claude-3-5-sonnet"] = ModelRate{InputPer1K: 0.003, OutputPer1K: 0.015, CachedPer1K: 0.0003}
	ci.rates["gpt-4o"] = ModelRate{InputPer1K: 0.0025, OutputPer1K: 0.010, CachedPer1K: 0.00125}

	return ci
}

// RegisterRate updates or adds a pricing rate for a model.
func (ci *Intelligence) RegisterRate(modelID string, inputPer1K, outputPer1K, cachedPer1K float64) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.rates[strings.ToLower(modelID)] = ModelRate{
		InputPer1K:  inputPer1K,
		OutputPer1K: outputPer1K,
		CachedPer1K: cachedPer1K,
	}
}

// MicroUSD represents millionths of a US Dollar (integer minor units: $1.00 = 1,000,000 MicroUSD).
type MicroUSD int64

// ToMicroUSD converts a USD float to integer MicroUSD, rejecting negative values, NaN, and infinity.
func ToMicroUSD(usd float64) (MicroUSD, error) {
	if usd < 0 {
		return 0, errors.New("monetary amount cannot be negative")
	}
	if math.IsNaN(usd) || math.IsInf(usd, 0) {
		return 0, errors.New("monetary amount cannot be NaN or Infinite")
	}
	return MicroUSD(math.Round(usd * 1_000_000.0)), nil
}

// ToUSD converts integer MicroUSD back to standard USD floating point representation.
func (m MicroUSD) ToUSD() float64 {
	return float64(m) / 1_000_000.0
}

// CalculateCost computes total USD cost based on token counts.
func (ci *Intelligence) CalculateCost(modelID string, inputTokens, outputTokens, cachedTokens int64) float64 {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	// Invariant: Reject negative token counts
	if inputTokens < 0 {
		inputTokens = 0
	}
	if outputTokens < 0 {
		outputTokens = 0
	}
	if cachedTokens < 0 {
		cachedTokens = 0
	}

	rate, exists := ci.rates[strings.ToLower(modelID)]
	if !exists {
		// Fallback default rate
		rate = ModelRate{InputPer1K: 0.001, OutputPer1K: 0.002}
	}

	inputCost := (float64(inputTokens) / 1000.0) * rate.InputPer1K
	outputCost := (float64(outputTokens) / 1000.0) * rate.OutputPer1K
	cachedCost := (float64(cachedTokens) / 1000.0) * rate.CachedPer1K

	total := inputCost + outputCost + cachedCost
	if math.IsNaN(total) || math.IsInf(total, 0) || total < 0 {
		return 0.0
	}
	return total
}

// CalculateCostMicroUSD computes cost directly as canonical integer MicroUSD.
func (ci *Intelligence) CalculateCostMicroUSD(modelID string, inputTokens, outputTokens, cachedTokens int64) MicroUSD {
	costUSD := ci.CalculateCost(modelID, inputTokens, outputTokens, cachedTokens)
	m, _ := ToMicroUSD(costUSD)
	return m
}

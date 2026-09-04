package cost

import (
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

// CalculateCost computes total USD cost based on token counts.
func (ci *Intelligence) CalculateCost(modelID string, inputTokens, outputTokens, cachedTokens int64) float64 {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	rate, exists := ci.rates[strings.ToLower(modelID)]
	if !exists {
		// Fallback default rate
		rate = ModelRate{InputPer1K: 0.001, OutputPer1K: 0.002}
	}

	inputCost := (float64(inputTokens) / 1000.0) * rate.InputPer1K
	outputCost := (float64(outputTokens) / 1000.0) * rate.OutputPer1K
	cachedCost := (float64(cachedTokens) / 1000.0) * rate.CachedPer1K

	return inputCost + outputCost + cachedCost
}

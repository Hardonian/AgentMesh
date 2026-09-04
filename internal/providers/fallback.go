package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ModelTarget describes a configured model endpoint and its operational health.
type ModelTarget struct {
	ModelID      string    `json:"modelId"`      // e.g. "gemini-1.5-pro", "gemini-1.5-flash"
	Provider     string    `json:"provider"`     // "gemini", "vertex", "local"
	Region       string    `json:"region"`       // "us-central1", "global"
	HealthStatus string    `json:"healthStatus"` // "HEALTHY", "DEGRADED", "UNAVAILABLE"
	P95LatencyMs int64     `json:"p95LatencyMs"`
	ErrorRate    float64   `json:"errorRate"`
	CostPer1kIn  float64   `json:"costPer1kIn"`
	CostPer1kOut float64   `json:"costPer1kOut"`
	LastChecked  time.Time `json:"lastChecked"`
}

// FallbackEvent logs whenever a model fails over to a secondary target.
type FallbackEvent struct {
	PrimaryModel   string    `json:"primaryModel"`
	FallbackModel  string    `json:"fallbackModel"`
	Reason         string    `json:"reason"`
	AllowedByPolicy bool     `json:"allowedByPolicy"`
	Timestamp      time.Time `json:"timestamp"`
}

// ModelRouter coordinates policy-governed model selection and safe fallback.
type ModelRouter struct {
	mu             sync.RWMutex
	targets        map[string]*ModelTarget
	fallbackEvents []FallbackEvent
	providers      map[string]ModelProvider
}

// NewModelRouter constructs a model routing and fallback coordinator.
func NewModelRouter() *ModelRouter {
	mr := &ModelRouter{
		targets:        make(map[string]*ModelTarget),
		fallbackEvents: make([]FallbackEvent, 0),
		providers:      make(map[string]ModelProvider),
	}

	// Register default targets
	mr.RegisterTarget(&ModelTarget{
		ModelID:      "gemini-1.5-pro",
		Provider:     "gemini",
		Region:       "global",
		HealthStatus: "HEALTHY",
		CostPer1kIn:  0.00125,
		CostPer1kOut: 0.00500,
	})
	mr.RegisterTarget(&ModelTarget{
		ModelID:      "gemini-1.5-flash",
		Provider:     "gemini",
		Region:       "global",
		HealthStatus: "HEALTHY",
		CostPer1kIn:  0.000075,
		CostPer1kOut: 0.000300,
	})

	mr.providers["gemini"] = NewGeminiProvider()
	mr.providers["vertex"] = NewVertexAIProvider()
	mr.providers["local"] = NewLocalDeterministicProvider()

	return mr
}

// RegisterTarget registers a model endpoint.
func (mr *ModelRouter) RegisterTarget(target *ModelTarget) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	target.LastChecked = time.Now().UTC()
	mr.targets[target.ModelID] = target
}

// GenerateWithFallback executes against the primary target, falling back only if allowed by policy.
func (mr *ModelRouter) GenerateWithFallback(ctx context.Context, primaryModel, fallbackModel string, allowedModels []string, req *GenerateRequest) (*GenerateResponse, *FallbackEvent, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	target, ok := mr.targets[primaryModel]
	providerName := "gemini"
	if ok && target.Provider != "" {
		providerName = target.Provider
	}
	prov := mr.providers[providerName]
	if prov == nil || prov.Status() == StateNotConfigured {
		prov = mr.providers["local"]
	}

	var resp *GenerateResponse
	var err error

	if ok && target.HealthStatus == "UNAVAILABLE" {
		err = fmt.Errorf("model target %s is UNAVAILABLE", primaryModel)
	} else {
		resp, err = prov.Generate(ctx, req)
	}

	if err == nil {
		return resp, nil, nil
	}

	// Primary failed: evaluate fallback permission
	if fallbackModel == "" {
		return nil, nil, fmt.Errorf("primary model %s failed and no fallback specified: %w", primaryModel, err)
	}

	// Verify if fallback model is allowed by model policy
	permitted := false
	if len(allowedModels) == 0 {
		permitted = true
	} else {
		for _, m := range allowedModels {
			if m == "*" || m == fallbackModel {
				permitted = true
				break
			}
		}
	}

	fbEvent := &FallbackEvent{
		PrimaryModel:    primaryModel,
		FallbackModel:   fallbackModel,
		Reason:          fmt.Sprintf("Primary failed: %v", err),
		AllowedByPolicy: permitted,
		Timestamp:       time.Now().UTC(),
	}
	mr.fallbackEvents = append(mr.fallbackEvents, *fbEvent)

	if !permitted {
		return nil, fbEvent, fmt.Errorf("model fallback to %s rejected by contract/policy", fallbackModel)
	}

	// Execute permitted fallback
	req.ModelID = fallbackModel
	fbTarget, fbOk := mr.targets[fallbackModel]
	fbProvName := "gemini"
	if fbOk && fbTarget.Provider != "" {
		fbProvName = fbTarget.Provider
	}
	fbProv := mr.providers[fbProvName]
	if fbProv == nil || fbProv.Status() == StateNotConfigured {
		fbProv = mr.providers["local"]
	}
	fbResp, fbErr := fbProv.Generate(ctx, req)
	if fbErr != nil {
		return nil, fbEvent, fmt.Errorf("fallback model %s also failed: %w", fallbackModel, fbErr)
	}

	return fbResp, fbEvent, nil
}

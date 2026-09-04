package learned

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ModelStatus defines the lifecycle phase of a routing model.
type ModelStatus string

const (
	StatusTraining  ModelStatus = "TRAINING"
	StatusCandidate ModelStatus = "CANDIDATE"
	StatusShadow    ModelStatus = "SHADOW"
	StatusActive    ModelStatus = "ACTIVE"
	StatusRetired   ModelStatus = "RETIRED"
)

// RoutingModelRecord describes a registered routing model artifact.
type RoutingModelRecord struct {
	ModelID               string             `json:"modelId"`
	TenantID              string             `json:"tenantId"`
	Version               string             `json:"version"`
	Status                ModelStatus        `json:"status"`
	DatasetSize           int                `json:"datasetSize"`
	SupportedCapabilities []string           `json:"supportedCapabilities"`
	FeatureWeights        map[string]float64 `json:"featureWeights"`
	AccuracyScore         float64            `json:"accuracyScore"`
	CostReductionPct      float64            `json:"costReductionPct"`
	CreatedAt             time.Time          `json:"createdAt"`
	PromotedAt            *time.Time         `json:"promotedAt,omitempty"`
}

// ModelRegistry manages routing model lifecycles, shadow evaluations, promotions, and rollbacks.
type ModelRegistry struct {
	mu           sync.RWMutex
	models       map[string]*RoutingModelRecord // tenant:modelId -> Model
	activeModel  map[string]string              // tenant -> modelId
	lastKnownGood map[string]string             // tenant -> modelId
}

// NewModelRegistry creates a new ModelRegistry.
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models:        make(map[string]*RoutingModelRecord),
		activeModel:   make(map[string]string),
		lastKnownGood: make(map[string]string),
	}
}

func modelKey(tenantID, modelID string) string {
	return tenantID + ":" + modelID
}

// RegisterModel registers a candidate routing model.
func (r *ModelRegistry) RegisterModel(record *RoutingModelRecord) error {
	if record == nil || record.ModelID == "" || record.TenantID == "" {
		return errors.New("invalid model record")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	k := modelKey(record.TenantID, record.ModelID)
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	if record.Status == "" {
		record.Status = StatusCandidate
	}
	r.models[k] = record
	return nil
}

// SetShadow places a model into shadow evaluation mode.
func (r *ModelRegistry) SetShadow(tenantID, modelID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	k := modelKey(tenantID, modelID)
	m, ok := r.models[k]
	if !ok {
		return fmt.Errorf("model %s not found", modelID)
	}
	m.Status = StatusShadow
	return nil
}

// Promote promotes a model to active status and updates last-known-good.
func (r *ModelRegistry) Promote(tenantID, modelID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	k := modelKey(tenantID, modelID)
	m, ok := r.models[k]
	if !ok {
		return fmt.Errorf("model %s not found", modelID)
	}

	// Archive previous active model if exists
	if prevActiveID, exists := r.activeModel[tenantID]; exists && prevActiveID != modelID {
		prevKey := modelKey(tenantID, prevActiveID)
		if prevModel, prevOk := r.models[prevKey]; prevOk {
			prevModel.Status = StatusRetired
			r.lastKnownGood[tenantID] = prevActiveID
		}
	}

	now := time.Now().UTC()
	m.Status = StatusActive
	m.PromotedAt = &now
	r.activeModel[tenantID] = modelID
	return nil
}

// Rollback restores the previous active model.
func (r *ModelRegistry) Rollback(tenantID string) (*RoutingModelRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lastGoodID, exists := r.lastKnownGood[tenantID]
	if !exists {
		return nil, errors.New("no last known good model recorded for rollback")
	}

	currentActiveID := r.activeModel[tenantID]
	if currentActiveID != "" {
		curKey := modelKey(tenantID, currentActiveID)
		if curModel, curOk := r.models[curKey]; curOk {
			curModel.Status = StatusRetired
		}
	}

	goodKey := modelKey(tenantID, lastGoodID)
	goodModel, ok := r.models[goodKey]
	if !ok {
		return nil, fmt.Errorf("last known good model %s missing from registry", lastGoodID)
	}

	now := time.Now().UTC()
	goodModel.Status = StatusActive
	goodModel.PromotedAt = &now
	r.activeModel[tenantID] = lastGoodID

	return goodModel, nil
}

// GetActiveModel returns the current production model for a tenant.
func (r *ModelRegistry) GetActiveModel(tenantID string) (*RoutingModelRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	activeID, exists := r.activeModel[tenantID]
	if !exists {
		return nil, false
	}
	m, ok := r.models[modelKey(tenantID, activeID)]
	return m, ok
}

// ListModels lists all models registered for a tenant.
func (r *ModelRegistry) ListModels(tenantID string) []*RoutingModelRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*RoutingModelRecord, 0)
	for _, m := range r.models {
		if tenantID == "" || m.TenantID == tenantID {
			list = append(list, m)
		}
	}
	return list
}

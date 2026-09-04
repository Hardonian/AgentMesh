package a2a

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// PublicCompatibilityEntry stores an anonymized public A2A compatibility observation.
type PublicCompatibilityEntry struct {
	AnonymousID     string            `json:"anonymousId"` // SHA-256 hash of runtime+salt
	Runtime         string            `json:"runtime"`     // "google-adk", "custom-go", "python"
	Framework       string            `json:"framework"`   // "adk-go", "langgraph", "crewai"
	ProtocolVersion string            `json:"protocolVersion"`
	OverallStatus   CompatibilityStatus `json:"overallStatus"` // "COMPATIBLE", "PARTIALLY_COMPATIBLE", "INCOMPATIBLE"
	Capabilities    map[string]string `json:"capabilities"`
	TesterVersion   string            `json:"testerVersion"`
	ObservedAt      time.Time         `json:"observedAt"`
}

// CompatibilityMatrixView aggregates verified results across runtimes and features.
type CompatibilityMatrixView struct {
	ProtocolVersion string                       `json:"protocolVersion"`
	Runtimes        []string                     `json:"runtimes"`
	Matrix          map[string]map[string]string `json:"matrix"` // Runtime -> Feature -> Status
	TotalEntries    int                          `json:"totalEntries"`
	UpdatedAt       time.Time                    `json:"updatedAt"`
}

// PublicCompatibilityRegistry manages sanitized public ecosystem compatibility observations.
type PublicCompatibilityRegistry struct {
	mu      sync.RWMutex
	entries map[string]*PublicCompatibilityEntry
}

// NewPublicCompatibilityRegistry constructs a registry.
func NewPublicCompatibilityRegistry() *PublicCompatibilityRegistry {
	return &PublicCompatibilityRegistry{
		entries: make(map[string]*PublicCompatibilityEntry),
	}
}

// PublishProfile sanitizes an internal profile and adds it to the public registry.
// Invariant: Never publishes private endpoints, tenant IDs, or private tool details.
func (r *PublicCompatibilityRegistry) PublishProfile(runtime, framework string, profile *A2ACompatibilityProfile) (*PublicCompatibilityEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if profile == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}

	// Generate anonymous ID to prevent correlation with private tenant identities
	salt := time.Now().UTC().Format("2006-01-02")
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s:%s", runtime, framework, profile.ProtocolVersion, salt)))
	anonID := hex.EncodeToString(hash[:16])

	entry := &PublicCompatibilityEntry{
		AnonymousID:     anonID,
		Runtime:         strings.ToLower(strings.TrimSpace(runtime)),
		Framework:       strings.ToLower(strings.TrimSpace(framework)),
		ProtocolVersion: profile.ProtocolVersion,
		OverallStatus:   profile.Status,
		Capabilities:    make(map[string]string),
		TesterVersion:   profile.TesterVersion,
		ObservedAt:      time.Now().UTC(),
	}

	for testName, res := range profile.Results {
		if res.Passed {
			entry.Capabilities[testName] = string(StatusCompatible)
		} else {
			entry.Capabilities[testName] = string(StatusIncompatible)
		}
	}

	r.entries[anonID] = entry
	return entry, nil
}

// GetMatrix computes an aggregated interoperability matrix.
func (r *PublicCompatibilityRegistry) GetMatrix(protocolVersion string) *CompatibilityMatrixView {
	r.mu.RLock()
	defer r.mu.RUnlock()

	view := &CompatibilityMatrixView{
		ProtocolVersion: protocolVersion,
		Runtimes:        make([]string, 0),
		Matrix:          make(map[string]map[string]string),
		UpdatedAt:       time.Now().UTC(),
	}

	runtimeSet := make(map[string]bool)

	for _, entry := range r.entries {
		if protocolVersion != "" && entry.ProtocolVersion != protocolVersion {
			continue
		}
		view.TotalEntries++
		runtimeKey := fmt.Sprintf("%s/%s", entry.Runtime, entry.Framework)
		runtimeSet[runtimeKey] = true

		if _, ok := view.Matrix[runtimeKey]; !ok {
			view.Matrix[runtimeKey] = make(map[string]string)
		}

		for capName, status := range entry.Capabilities {
			view.Matrix[runtimeKey][capName] = status
		}
	}

	for rt := range runtimeSet {
		view.Runtimes = append(view.Runtimes, rt)
	}

	return view
}

// ListEntries returns all sanitized public records.
func (r *PublicCompatibilityRegistry) ListEntries() []*PublicCompatibilityEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*PublicCompatibilityEntry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	return out
}

package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// GoldenTask represents a single reference task within a golden task set.
type GoldenTask struct {
	TaskID              string            `json:"taskId"`
	CapabilityID        string            `json:"capabilityId"`
	SanitizedInput      map[string]any    `json:"sanitizedInput"`
	ExpectedOutputMatch string            `json:"expectedOutputMatch,omitempty"`
	MaxAllowedLatencyMs int64             `json:"maxAllowedLatencyMs"`
	MaxAllowedCostUSD   float64           `json:"maxAllowedCostUsd"`
	PermittedTools      []string          `json:"permittedTools"`
	QualityAssertions   map[string]any    `json:"qualityAssertions,omitempty"`
}

// GoldenTaskSet defines a tenant-private, versioned test suite used before canaries.
type GoldenTaskSet struct {
	ID             string       `json:"id"`
	OrganizationID string       `json:"organizationId"`
	ProjectID      string       `json:"projectId"`
	CapabilityID   string       `json:"capabilityId"`
	Version        string       `json:"version"`
	Tasks          []GoldenTask `json:"tasks"`
	CreatedAt      time.Time    `json:"createdAt"`
	UpdatedAt      time.Time    `json:"updatedAt"`
}

// ComputeDigest computes a deterministic SHA-256 digest of the task set.
func (g *GoldenTaskSet) ComputeDigest() string {
	h := sha256.New()
	content := fmt.Sprintf("%s:%s:%s:%d:%d",
		g.OrganizationID,
		g.CapabilityID,
		g.Version,
		len(g.Tasks),
		g.CreatedAt.Unix(),
	)
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

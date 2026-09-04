package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/internal/routing"
)

// BigQueryExportBatch represents a formatted export payload ready for streaming or batch ingestion into BigQuery.
type BigQueryExportBatch struct {
	TenantID     string                            `json:"tenantId"`
	ProjectID    string                            `json:"projectId"`
	DatasetID    string                            `json:"datasetId"`
	TableName    string                            `json:"tableName"`
	ExportedAt   time.Time                         `json:"exportedAt"`
	RecordCount  int                               `json:"recordCount"`
	OutcomeRows  []*routing.CanonicalRoutingOutcome `json:"outcomeRows"`
}

// Exporter manages formatting and streaming of outcomes to tenant-isolated BigQuery datasets.
type Exporter struct {
	mu           sync.RWMutex
	exportedBatches []*BigQueryExportBatch
}

// NewExporter creates a new analytics exporter.
func NewExporter() *Exporter {
	return &Exporter{
		exportedBatches: make([]*BigQueryExportBatch, 0),
	}
}

// FormatDatasetName generates a deterministic, tenant-isolated dataset identifier.
func FormatDatasetName(tenantID string) string {
	clean := ""
	for _, ch := range tenantID {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			clean += string(ch)
		} else {
			clean += "_"
		}
	}
	return "agentmesh_analytics_" + clean
}

// ExportBatch creates a tenant-isolated BigQuery export bundle.
func (e *Exporter) ExportBatch(
	ctx context.Context,
	tenantID, gcpProject string,
	outcomes []*routing.CanonicalRoutingOutcome,
) (*BigQueryExportBatch, error) {
	if tenantID == "" {
		return nil, errors.New("tenantId is required")
	}
	if gcpProject == "" {
		gcpProject = "agentmesh-analytics-default"
	}

	datasetID := FormatDatasetName(tenantID)

	// Filter strictly by tenantID to guarantee zero cross-tenant contamination
	filtered := make([]*routing.CanonicalRoutingOutcome, 0, len(outcomes))
	for _, o := range outcomes {
		if o.OrganizationID == tenantID {
			filtered = append(filtered, o)
		}
	}

	batch := &BigQueryExportBatch{
		TenantID:    tenantID,
		ProjectID:   gcpProject,
		DatasetID:   datasetID,
		TableName:   "routing_outcomes_v3",
		ExportedAt:  time.Now().UTC(),
		RecordCount: len(filtered),
		OutcomeRows: filtered,
	}

	e.mu.Lock()
	e.exportedBatches = append(e.exportedBatches, batch)
	e.mu.Unlock()

	return batch, nil
}

// ToJSONLines serializes the batch into newline-delimited JSON for BigQuery load jobs.
func (b *BigQueryExportBatch) ToJSONLines() ([]byte, error) {
	var buf []byte
	for _, row := range b.OutcomeRows {
		line, err := json.Marshal(row)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize outcome row: %w", err)
		}
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}
	return buf, nil
}

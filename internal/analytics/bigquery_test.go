package analytics

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/routing"
)

func TestBigQueryExportTenantIsolation(t *testing.T) {
	exp := NewExporter()
	ctx := context.Background()

	outcomes := []*routing.CanonicalRoutingOutcome{
		{
			OutcomeID:      "out-1",
			OrganizationID: "tenant-a",
			TaskID:         "task-1",
			Success:        true,
			Cost:           0.05,
			CreatedAt:      time.Now().UTC(),
		},
		{
			OutcomeID:      "out-2",
			OrganizationID: "tenant-b", // Another tenant!
			TaskID:         "task-2",
			Success:        false,
			Cost:           0.02,
			CreatedAt:      time.Now().UTC(),
		},
	}

	// Export for tenant-a
	batchA, err := exp.ExportBatch(ctx, "tenant-a", "my-gcp-project", outcomes)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	if batchA.DatasetID != "agentmesh_analytics_tenant_a" {
		t.Errorf("expected dataset agentmesh_analytics_tenant_a, got %s", batchA.DatasetID)
	}
	if batchA.RecordCount != 1 {
		t.Fatalf("expected exactly 1 record for tenant-a, got %d", batchA.RecordCount)
	}
	if batchA.OutcomeRows[0].OutcomeID != "out-1" {
		t.Errorf("expected outcome out-1, got %s", batchA.OutcomeRows[0].OutcomeID)
	}

	// Test JSON Lines serialization
	jsonl, err := batchA.ToJSONLines()
	if err != nil {
		t.Fatalf("jsonl serialization failed: %v", err)
	}
	if !strings.Contains(string(jsonl), `"outcome_id":"out-1"`) {
		t.Errorf("jsonl missing out-1: %s", string(jsonl))
	}
}

package evaluation_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/agentmesh/agentmesh/internal/evaluation"
)

func TestEvaluationSuiteExecution(t *testing.T) {
	suite := &evaluation.EvaluationSuite{
		ID:         "suite_finance",
		Capability: "invoice_analysis",
		TestCases: []evaluation.EvaluationTestCase{
			{
				ID:             "tc_1",
				Name:           "Valid Invoice",
				ExpectedFields: []string{"total", "vendor"},
				AllowedTools:   []string{"bigquery.read"},
				ForbiddenTools: []string{"payment.execute"},
				MaxLatencyMs:   2000,
			},
		},
	}

	report, prov, err := suite.ExecuteSuite(context.Background(), "finance-agent", "1.0.0", "gemini-1.5-pro", func(ctx context.Context, tc evaluation.EvaluationTestCase) (output map[string]any, toolsUsed []string, latencyMs int64, costUSD float64, err error) {
		return map[string]any{
			"total":  150.00,
			"vendor": "Acme Supplies",
		}, []string{"bigquery.read"}, 450, 0.02, nil
	})

	if err != nil {
		t.Fatalf("failed to execute evaluation suite: %v", err)
	}
	if report.OverallScore != 1.0 {
		t.Fatalf("expected perfect score 1.0, got %f", report.OverallScore)
	}
	if prov.Method != evaluation.MethodDeterministic {
		t.Fatalf("expected deterministic method, got %s", prov.Method)
	}
}

func TestPerformanceCIConfig(t *testing.T) {
	yml := `version: 1
agent:
  name: finance-agent
evaluation:
  suite: finance-regression
thresholds:
  min_quality: 0.90
  max_p95_latency_ms: 3000
  max_cost_per_task: 0.05
policy:
  require_pass: true
canary:
  enabled: true
`
	tmpDir, err := os.MkdirTemp("", "ci-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".agentmesh.yml")
	if err := os.WriteFile(cfgPath, []byte(yml), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := evaluation.LoadCIConfig(cfgPath)
	if err != nil {
		t.Fatalf("failed to load CI config: %v", err)
	}

	// Test passing scenario
	repPass := cfg.EvaluateCIPerformance("finance-agent", "1.1.0", 0.95, 1200, 0.03, 1.0, true)
	if !repPass.SafeToCanary {
		t.Fatalf("expected safeToCanary to be true when metrics satisfy thresholds")
	}

	// Test failing scenario (latency exceeded)
	repFail := cfg.EvaluateCIPerformance("finance-agent", "1.1.0", 0.95, 4500, 0.03, 1.0, true)
	if repFail.SafeToCanary {
		t.Fatalf("expected safeToCanary to be false when latency breaches max")
	}
}

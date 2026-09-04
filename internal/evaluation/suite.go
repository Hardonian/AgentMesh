package evaluation

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// EvaluationMethod specifies how the output is verified.
type EvaluationMethod string

const (
	MethodDeterministic EvaluationMethod = "DETERMINISTIC"
	MethodLLMJudge      EvaluationMethod = "LLM_AS_JUDGE"
	MethodHybrid        EvaluationMethod = "HYBRID"
)

// LLMJudgeConfig configures optional model-graded evaluations.
type LLMJudgeConfig struct {
	Enabled      bool    `json:"enabled" yaml:"enabled"`
	JudgeModel   string  `json:"judgeModel" yaml:"judge_model"` // e.g. "gemini-1.5-pro"
	JudgeVersion string  `json:"judgeVersion" yaml:"judge_version"`
	Rubric       string  `json:"rubric" yaml:"rubric"`
	Temperature  float64 `json:"temperature" yaml:"temperature"`
}

// EvaluationTestCase defines rich execution constraints and assertions.
type EvaluationTestCase struct {
	ID             string         `json:"id" yaml:"id"`
	Name           string         `json:"name" yaml:"name"`
	Input          map[string]any `json:"input" yaml:"input"`
	ExpectedFields []string       `json:"expectedFields,omitempty" yaml:"expected_fields,omitempty"`
	AllowedTools   []string       `json:"allowedTools,omitempty" yaml:"allowed_tools,omitempty"`
	ForbiddenTools []string       `json:"forbiddenTools,omitempty" yaml:"forbidden_tools,omitempty"`
	MaxCostUSD     float64        `json:"maxCostUsd,omitempty" yaml:"max_cost_usd,omitempty"`
	MaxLatencyMs   int64          `json:"maxLatencyMs,omitempty" yaml:"max_latency_ms,omitempty"`
}

// EvaluationSuite specifies a repeatable test suite bound to an agent capability.
type EvaluationSuite struct {
	ID           string               `json:"id" yaml:"id"`
	TenantID     string               `json:"tenantId" yaml:"tenant_id"`
	Capability   string               `json:"capability" yaml:"capability"`
	Description  string               `json:"description" yaml:"description"`
	TestCases    []EvaluationTestCase `json:"testCases" yaml:"test_cases"`
	LLMJudge     *LLMJudgeConfig      `json:"llmJudge,omitempty" yaml:"llm_judge,omitempty"`
	CreatedAt    time.Time            `json:"createdAt"`
}

// EvaluationProvenance details the exact conditions under which a score was produced.
type EvaluationProvenance struct {
	Method         EvaluationMethod `json:"method"`
	JudgeModel     string           `json:"judgeModel,omitempty"`
	JudgeVersion   string           `json:"judgeVersion,omitempty"`
	RubricVersion  string           `json:"rubricVersion,omitempty"`
	AgentID        string           `json:"agentId"`
	AgentVersion   string           `json:"agentVersion"`
	ModelTarget    string           `json:"modelTarget"`
	SuiteID        string           `json:"suiteId"`
	TestSetVersion string           `json:"testSetVersion"`
	EvaluatedAt    time.Time        `json:"evaluatedAt"`
}

// EvaluationBaseline represents an immutable historical benchmark for regression detection.
type EvaluationBaseline struct {
	ID           string               `json:"id"`
	TenantID     string               `json:"tenantId"`
	AgentID      string               `json:"agentId"`
	Version      string               `json:"version"`
	SuiteID      string               `json:"suiteId"`
	QualityScore float64              `json:"qualityScore"`
	P95LatencyMs int64                `json:"p95LatencyMs"`
	CostUSD      float64              `json:"costUsd"`
	Provenance   EvaluationProvenance `json:"provenance"`
	CreatedAt    time.Time            `json:"createdAt"`
}

// ExecuteSuite runs the test suite and returns scored results with provenance.
func (s *EvaluationSuite) ExecuteSuite(ctx context.Context, agentID, version, modelTarget string, invokeFn func(ctx context.Context, tc EvaluationTestCase) (output map[string]any, toolsUsed []string, latencyMs int64, costUSD float64, err error)) (*SuiteReport, *EvaluationProvenance, error) {
	report := &SuiteReport{
		AgentID:     agentID,
		Version:     version,
		TotalTests:  len(s.TestCases),
		Results:     make([]TestResult, 0, len(s.TestCases)),
		EvaluatedAt: time.Now().UTC(),
	}

	totalScore := 0.0
	for _, tc := range s.TestCases {
		res := TestResult{
			TestCaseID: tc.ID,
			Status:     StatusPass,
		}

		output, toolsUsed, latencyMs, costUSD, err := invokeFn(ctx, tc)
		res.CostUSD = costUSD
		res.Duration = time.Duration(latencyMs) * time.Millisecond

		if err != nil {
			res.Status = StatusFail
			res.Reason = fmt.Sprintf("execution failed: %v", err)
			report.FailedTests++
			report.Results = append(report.Results, res)
			continue
		}

		// 1. Tool assertions
		for _, used := range toolsUsed {
			for _, forbidden := range tc.ForbiddenTools {
				if strings.EqualFold(used, forbidden) {
					res.Status = StatusFail
					res.Reason = fmt.Sprintf("forbidden tool %q was invoked", used)
					break
				}
			}
			if res.Status == StatusFail {
				break
			}
		}

		// 2. Latency target
		if tc.MaxLatencyMs > 0 && latencyMs > tc.MaxLatencyMs {
			res.Status = StatusFail
			res.Reason = fmt.Sprintf("latency %dms exceeded target %dms", latencyMs, tc.MaxLatencyMs)
		}

		// 3. Expected fields in output
		for _, expField := range tc.ExpectedFields {
			if _, ok := output[expField]; !ok {
				res.Status = StatusFail
				res.Reason = fmt.Sprintf("missing expected field %q in output", expField)
				break
			}
		}

		if res.Status == StatusPass {
			report.PassedTests++
			totalScore += 1.0
		} else {
			report.FailedTests++
		}
		report.Results = append(report.Results, res)
	}

	if report.TotalTests > 0 {
		report.OverallScore = totalScore / float64(report.TotalTests)
	}
	if report.FailedTests == 0 {
		report.Status = StatusPass
	} else {
		report.Status = StatusFail
	}

	method := MethodDeterministic
	judgeModel := ""
	if s.LLMJudge != nil && s.LLMJudge.Enabled {
		method = MethodHybrid
		judgeModel = s.LLMJudge.JudgeModel
	}

	prov := &EvaluationProvenance{
		Method:         method,
		JudgeModel:     judgeModel,
		AgentID:        agentID,
		AgentVersion:   version,
		ModelTarget:    modelTarget,
		SuiteID:        s.ID,
		TestSetVersion: "1.0.0",
		EvaluatedAt:    report.EvaluatedAt,
	}

	return report, prov, nil
}

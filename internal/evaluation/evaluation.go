package evaluation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	StatusPass         Status = "PASS"
	StatusFail         Status = "FAIL"
	StatusError        Status = "ERROR"
	StatusInconclusive Status = "INCONCLUSIVE"
)

// TestCase specifies a single evaluation scenario.
type TestCase struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	InputCapability    string            `json:"inputCapability"`
	InputParameters    map[string]any    `json:"inputParameters"`
	ExpectedSubstrings []string          `json:"expectedSubstrings,omitempty"`
	ForbiddenSubstrings []string         `json:"forbiddenSubstrings,omitempty"`
	MaxAllowedLatencyMs int64            `json:"maxAllowedLatencyMs,omitempty"`
	MaxAllowedCostUSD   float64          `json:"maxAllowedCostUsd,omitempty"`
}

// TestResult records the outcome of a single test case execution.
type TestResult struct {
	TestCaseID string        `json:"testCaseId"`
	Status     Status        `json:"status"`
	Duration   time.Duration `json:"duration"`
	CostUSD    float64       `json:"costUsd"`
	Reason     string        `json:"reason"`
}

// SuiteReport aggregates the results of an entire evaluation run.
type SuiteReport struct {
	AgentID       string        `json:"agentId"`
	Version       string        `json:"version"`
	TotalTests    int           `json:"totalTests"`
	PassedTests   int           `json:"passedTests"`
	FailedTests   int           `json:"failedTests"`
	OverallScore  float64       `json:"overallScore"` // 0.0 to 1.0
	Status        Status        `json:"status"`
	Duration      time.Duration `json:"duration"`
	Results       []TestResult  `json:"results"`
	EvaluatedAt   time.Time     `json:"evaluatedAt"`
}

// RegressionReport compares candidate against baseline.
type RegressionReport struct {
	AgentID               string  `json:"agentId"`
	BaselineVersion       string  `json:"baselineVersion"`
	CandidateVersion      string  `json:"candidateVersion"`
	BaselineScore         float64 `json:"baselineScore"`
	CandidateScore        float64 `json:"candidateScore"`
	ScoreDelta            float64 `json:"scoreDelta"`
	HasRegression         bool    `json:"hasRegression"`
	RegressionDescription string  `json:"regressionDescription,omitempty"`
}

// Evaluator runs test suites against agents.
type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// ExecuteSuite runs test cases against an execution function.
func (e *Evaluator) ExecuteSuite(ctx context.Context, agentID, version string, tests []TestCase, runFn func(ctx context.Context, tc *TestCase) (output string, cost float64, err error)) *SuiteReport {
	start := time.Now().UTC()
	report := &SuiteReport{
		AgentID:     agentID,
		Version:     version,
		TotalTests:  len(tests),
		EvaluatedAt: start,
		Results:     make([]TestResult, 0, len(tests)),
	}

	for _, tc := range tests {
		tcStart := time.Now()
		out, cost, err := runFn(ctx, &tc)
		duration := time.Since(tcStart)

		res := TestResult{
			TestCaseID: tc.ID,
			Duration:   duration,
			CostUSD:    cost,
		}

		if err != nil {
			res.Status = StatusError
			res.Reason = fmt.Sprintf("execution error: %v", err)
		} else {
			// Validate assertions
			passed := true
			var failureReasons []string

			for _, sub := range tc.ExpectedSubstrings {
				if !strings.Contains(out, sub) {
					passed = false
					failureReasons = append(failureReasons, fmt.Sprintf("missing expected substring %q", sub))
				}
			}

			for _, forbi := range tc.ForbiddenSubstrings {
				if strings.Contains(out, forbi) {
					passed = false
					failureReasons = append(failureReasons, fmt.Sprintf("contains forbidden substring %q", forbi))
				}
			}

			if tc.MaxAllowedLatencyMs > 0 && duration.Milliseconds() > tc.MaxAllowedLatencyMs {
				passed = false
				failureReasons = append(failureReasons, fmt.Sprintf("latency %dms exceeded limit %dms", duration.Milliseconds(), tc.MaxAllowedLatencyMs))
			}

			if tc.MaxAllowedCostUSD > 0 && cost > tc.MaxAllowedCostUSD {
				passed = false
				failureReasons = append(failureReasons, fmt.Sprintf("cost $%.4f exceeded ceiling $%.4f", cost, tc.MaxAllowedCostUSD))
			}

			if passed {
				res.Status = StatusPass
				res.Reason = "all assertions satisfied"
				report.PassedTests++
			} else {
				res.Status = StatusFail
				res.Reason = strings.Join(failureReasons, "; ")
				report.FailedTests++
			}
		}

		report.Results = append(report.Results, res)
	}

	report.Duration = time.Since(start)
	if report.TotalTests > 0 {
		report.OverallScore = float64(report.PassedTests) / float64(report.TotalTests)
	}

	if report.FailedTests == 0 && report.TotalTests > 0 {
		report.Status = StatusPass
	} else {
		report.Status = StatusFail
	}

	return report
}

// CompareForRegression checks if candidate regressed compared to baseline.
func CompareForRegression(baseline, candidate *SuiteReport, tolerance float64) (*RegressionReport, error) {
	if baseline == nil || candidate == nil {
		return nil, errors.New("reports cannot be nil")
	}

	delta := candidate.OverallScore - baseline.OverallScore
	hasReg := false
	desc := ""

	if candidate.OverallScore < (baseline.OverallScore - tolerance) {
		hasReg = true
		desc = fmt.Sprintf("score regressed by %.1f%% (baseline: %.1f%%, candidate: %.1f%%)",
			-delta*100, baseline.OverallScore*100, candidate.OverallScore*100)
	}

	return &RegressionReport{
		AgentID:               candidate.AgentID,
		BaselineVersion:       baseline.Version,
		CandidateVersion:      candidate.Version,
		BaselineScore:         baseline.OverallScore,
		CandidateScore:        candidate.OverallScore,
		ScoreDelta:            delta,
		HasRegression:         hasReg,
		RegressionDescription: desc,
	}, nil
}

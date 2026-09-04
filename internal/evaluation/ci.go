package evaluation

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CIConfig maps directly to the strict schema of .agentmesh.yml.
type CIConfig struct {
	Version int `json:"version" yaml:"version"`
	Agent   struct {
		Name string `json:"name" yaml:"name"`
	} `json:"agent" yaml:"agent"`
	Evaluation struct {
		Suite    string `json:"suite" yaml:"suite"`
		Baseline string `json:"baseline,omitempty" yaml:"baseline,omitempty"`
	} `json:"evaluation" yaml:"evaluation"`
	Thresholds struct {
		MinQuality      float64 `json:"min_quality" yaml:"min_quality"`
		MaxP95LatencyMs int64   `json:"max_p95_latency_ms" yaml:"max_p95_latency_ms"`
		MaxCostPerTask  float64 `json:"max_cost_per_task" yaml:"max_cost_per_task"`
	} `json:"thresholds" yaml:"thresholds"`
	Policy struct {
		RequirePass bool `json:"require_pass" yaml:"require_pass"`
	} `json:"policy" yaml:"policy"`
	Canary struct {
		Enabled bool `json:"enabled" yaml:"enabled"`
	} `json:"canary" yaml:"canary"`
}

// LoadCIConfig parses a .agentmesh.yml file from disk.
func LoadCIConfig(path string) (*CIConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ci config %s: %w", path, err)
	}

	var cfg CIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse ci config yaml: %w", err)
	}

	if cfg.Version != 1 {
		return nil, fmt.Errorf("unsupported .agentmesh.yml version %d, expected 1", cfg.Version)
	}
	if cfg.Agent.Name == "" {
		return nil, errors.New("agent.name is required in .agentmesh.yml")
	}
	if cfg.Evaluation.Suite == "" {
		return nil, errors.New("evaluation.suite is required in .agentmesh.yml")
	}

	return &cfg, nil
}

// PerformanceCIReport summarizes evaluation metrics against CI threshold gates.
type PerformanceCIReport struct {
	AgentID         string  `json:"agentId"`
	Version         string  `json:"version"`
	QualityScore    float64 `json:"qualityScore"`
	QualityPass     bool    `json:"qualityPass"`
	P95LatencyMs    int64   `json:"p95LatencyMs"`
	LatencyPass     bool    `json:"latencyPass"`
	CostUSD         float64 `json:"costUsd"`
	CostPass        bool    `json:"costPass"`
	PolicyPass      bool    `json:"policyPass"`
	ToolSuccessRate float64 `json:"toolSuccessRate"`
	SafeToCanary    bool    `json:"safeToCanary"`
	Summary         string  `json:"summary"`
}

// EvaluateCIPerformance evaluates an agent's run metrics against .agentmesh.yml thresholds.
func (cfg *CIConfig) EvaluateCIPerformance(agentID, version string, qualityScore float64, p95LatencyMs int64, costUSD float64, toolSuccessRate float64, policyPass bool) *PerformanceCIReport {
	rep := &PerformanceCIReport{
		AgentID:         agentID,
		Version:         version,
		QualityScore:    qualityScore,
		P95LatencyMs:    p95LatencyMs,
		CostUSD:         costUSD,
		ToolSuccessRate: toolSuccessRate,
		PolicyPass:      policyPass,
	}

	rep.QualityPass = qualityScore >= cfg.Thresholds.MinQuality
	rep.LatencyPass = cfg.Thresholds.MaxP95LatencyMs == 0 || p95LatencyMs <= cfg.Thresholds.MaxP95LatencyMs
	rep.CostPass = cfg.Thresholds.MaxCostPerTask == 0 || costUSD <= cfg.Thresholds.MaxCostPerTask

	policyGate := true
	if cfg.Policy.RequirePass && !policyPass {
		policyGate = false
	}

	rep.SafeToCanary = rep.QualityPass && rep.LatencyPass && rep.CostPass && policyGate

	if rep.SafeToCanary {
		rep.Summary = fmt.Sprintf("All CI gates PASSED. Agent %s (v%s) is safe to canary.", agentID, version)
	} else {
		rep.Summary = fmt.Sprintf("CI gates FAILED for agent %s (v%s). Review thresholds before canary promotion.", agentID, version)
	}

	return rep
}

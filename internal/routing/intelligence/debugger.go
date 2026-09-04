package intelligence

import (
	"fmt"
	"time"

	"github.com/agentmesh/agentmesh/pkg/task"
)

// RouteDebugReport reconstructs all context and decision reasoning for audit and CLI debugging.
type RouteDebugReport struct {
	TaskID              string                `json:"taskId"`
	CapabilityID        string                `json:"capabilityId"`
	SelectedAgentID     string                `json:"selectedAgentId"`
	Objective           RoutingObjective      `json:"objective"`
	AlgorithmID         string                `json:"algorithmId"`
	PolicyVersion       string                `json:"policyVersion"`
	DecidedAt           time.Time             `json:"decidedAt"`
	DecisionExplanation string                `json:"decisionExplanation"`
	TaskFingerprint     *task.TaskFingerprint `json:"taskFingerprint"`
	EligibleCandidates  []*ScoredCandidate    `json:"eligibleCandidates"`
	ExcludedCandidates  []*ScoredCandidate    `json:"excludedCandidates"`
	EvidenceSummary     string                `json:"evidenceSummary"`
}

// BuildDebugReport constructs the complete debug payload from a RouteResult and TaskFingerprint.
func BuildDebugReport(taskID string, fp *task.TaskFingerprint, res *RouteResult) *RouteDebugReport {
	report := &RouteDebugReport{
		TaskID:              taskID,
		CapabilityID:        fp.Capability,
		SelectedAgentID:     res.SelectedAgentID,
		Objective:           res.Objective,
		AlgorithmID:         res.AlgorithmID + " v" + res.AlgorithmVersion,
		PolicyVersion:       res.PolicyVersion,
		DecidedAt:           res.DecidedAt,
		DecisionExplanation: res.DecisionExplanation,
		TaskFingerprint:     fp,
		EligibleCandidates:  make([]*ScoredCandidate, 0),
		ExcludedCandidates:  make([]*ScoredCandidate, 0),
	}

	for _, c := range res.Candidates {
		if c.Eligible {
			report.EligibleCandidates = append(report.EligibleCandidates, c)
		} else {
			report.ExcludedCandidates = append(report.ExcludedCandidates, c)
		}
	}

	report.EvidenceSummary = fmt.Sprintf(
		"Total Candidates Evaluated: %d | Eligible: %d | Excluded: %d | Final Decision: %s",
		len(res.Candidates),
		len(report.EligibleCandidates),
		len(report.ExcludedCandidates),
		res.SelectedAgentID,
	)

	return report
}

package reconcile

import (
	"fmt"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// ImpactDimension flags specific changes.
type ImpactDimension struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
}

// ChangeImpactReport provides an objective breakdown of proposed mutations.
type ChangeImpactReport struct {
	RiskClass            spec.RiskClass    `json:"riskClass"`
	Dimensions           []ImpactDimension `json:"dimensions"`
	BlastRadius          spec.BlastRadius  `json:"blastRadius"`
	RequiresHumanApproval bool              `json:"requiresHumanApproval"`
	ApprovalReasons      []string          `json:"approvalReasons"`
}

// AnalyzeActionImpact inspects proposed state changes against risk heuristics.
func AnalyzeActionImpact(action *spec.AgentOptimizationAction) *ChangeImpactReport {
	report := &ChangeImpactReport{
		RiskClass:            spec.RiskLow,
		Dimensions:           make([]ImpactDimension, 0),
		BlastRadius:          action.BlastRadius,
		RequiresHumanApproval: false,
		ApprovalReasons:      make([]string, 0),
	}

	switch action.ActionType {
	case spec.ActionChangeRouteWeight:
		// Shifting route weight
		currWeight, _ := action.CurrentState["weight"].(int)
		propWeight, _ := action.ProposedState["weight"].(int)
		delta := propWeight - currWeight
		if delta < 0 {
			delta = -delta
		}

		if delta > 50 {
			report.RiskClass = spec.RiskMedium
			report.Dimensions = append(report.Dimensions, ImpactDimension{
				Name:        "large_weight_shift",
				Description: fmt.Sprintf("Traffic weight delta (%d%%) is greater than 50%%", delta),
				Severity:    "MEDIUM",
			})
		} else {
			report.RiskClass = spec.RiskLow
			report.Dimensions = append(report.Dimensions, ImpactDimension{
				Name:        "minor_weight_shift",
				Description: fmt.Sprintf("Traffic weight delta (%d%%) is low risk", delta),
				Severity:    "LOW",
			})
		}

	case spec.ActionChangeModelTarget:
		// Changing underlying model (e.g. Gemini-1.5-Flash to Gemini-1.5-Pro)
		currModel, _ := action.CurrentState["model"].(string)
		propModel, _ := action.ProposedState["model"].(string)
		report.RiskClass = spec.RiskHigh
		report.RequiresHumanApproval = true
		report.ApprovalReasons = append(report.ApprovalReasons, "model target change carries behavioral risk")
		report.Dimensions = append(report.Dimensions, ImpactDimension{
			Name:        "model_target_change",
			Description: fmt.Sprintf("Model changing from %s to %s", currModel, propModel),
			Severity:    "HIGH",
		})

	case spec.ActionChangeAgentVersion:
		report.RiskClass = spec.RiskMedium
		report.Dimensions = append(report.Dimensions, ImpactDimension{
			Name:        "agent_version_change",
			Description: "Agent revision update",
			Severity:    "MEDIUM",
		})

	case spec.ActionChangeToolProvider:
		// Changing tool provider might involve write or destructive tools
		toolRisk, _ := action.ProposedState["toolRiskClass"].(string)
		if toolRisk == "DESTRUCTIVE" || toolRisk == "FINANCIAL" {
			report.RiskClass = spec.RiskCritical
			report.RequiresHumanApproval = true
			report.ApprovalReasons = append(report.ApprovalReasons, "tool provider involves destructive/financial capabilities")
			report.Dimensions = append(report.Dimensions, ImpactDimension{
				Name:        "privileged_tool_provider",
				Description: "Tool provider update has critical risk class",
				Severity:    "CRITICAL",
			})
		} else {
			report.RiskClass = spec.RiskMedium
			report.Dimensions = append(report.Dimensions, ImpactDimension{
				Name:        "tool_provider_change",
				Description: "Standard tool provider substitution",
				Severity:    "MEDIUM",
			})
		}

	case spec.ActionDisableAgent:
		report.RiskClass = spec.RiskHigh
		report.Dimensions = append(report.Dimensions, ImpactDimension{
			Name:        "agent_disablement",
			Description: "Disabling an agent reduces available capability pool",
			Severity:    "HIGH",
		})

	case spec.ActionRollbackAgentVersion:
		// Rollbacks are inherently safety-restoring
		report.RiskClass = spec.RiskLow
		report.Dimensions = append(report.Dimensions, ImpactDimension{
			Name:        "agent_rollback",
			Description: "Restoring prior known-good version",
			Severity:    "LOW",
		})

	default:
		report.RiskClass = spec.RiskMedium
	}

	// Blast radius checks
	if action.BlastRadius.TrafficPercent > 50 {
		if report.RiskClass == spec.RiskLow {
			report.RiskClass = spec.RiskMedium
		}
		report.Dimensions = append(report.Dimensions, ImpactDimension{
			Name:        "high_traffic_blast_radius",
			Description: fmt.Sprintf("Affects %d%% of capability traffic", action.BlastRadius.TrafficPercent),
			Severity:    "HIGH",
		})
	}

	return report
}

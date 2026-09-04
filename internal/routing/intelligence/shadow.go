package intelligence

import (
	"context"
	"strings"

	"github.com/agentmesh/agentmesh/pkg/task"
)

// ShadowRouteResult packages primary decision alongside non-interfering shadow decision.
type ShadowRouteResult struct {
	PrimaryRoute          *RouteResult `json:"primaryRoute"`
	ShadowRoute           *RouteResult `json:"shadowRoute"`
	DecisionsAgree        bool         `json:"decisionsAgree"`
	SideEffectsSuppressed bool         `json:"sideEffectsSuppressed"`
	SuppressionReason     string       `json:"suppressionReason,omitempty"`
}

// IsToolDestructive checks whether a tool name implies write or destructive capability.
func IsToolDestructive(toolName string) bool {
	lower := strings.ToLower(toolName)
	destructiveKeywords := []string{
		"write", "delete", "drop", "update", "insert", "create",
		"payment", "send", "transfer", "publish", "execute", "destroy",
	}
	for _, kw := range destructiveKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// EvaluateShadowRoute computes the shadow route with strict side-effect suppression.
func EvaluateShadowRoute(
	ctx context.Context,
	req *task.TaskFingerprint,
	primary *RouteResult,
	shadowRouter *BaselineRouterV1,
	tenantID string,
	candidates []*CandidateAgent,
	policyVersion string,
) (*ShadowRouteResult, error) {
	shadowRes, err := shadowRouter.Route(ctx, req, tenantID, ObjectiveBalanced, nil, candidates, policyVersion)
	if err != nil {
		return nil, err
	}

	result := &ShadowRouteResult{
		PrimaryRoute:   primary,
		ShadowRoute:    shadowRes,
		DecisionsAgree: (primary.SelectedAgentID == shadowRes.SelectedAgentID),
	}

	// Verify Side-Effect Suppression (Section 14)
	hasDestructiveTools := false
	for _, t := range req.RequiredTools {
		if IsToolDestructive(t) {
			hasDestructiveTools = true
			break
		}
	}

	if hasDestructiveTools {
		result.SideEffectsSuppressed = true
		result.SuppressionReason = "Shadow execution suppressed: task requires write or external communication tools"
	} else {
		result.SideEffectsSuppressed = false
	}

	return result, nil
}

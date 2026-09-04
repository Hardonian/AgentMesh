package routing

import (
	"fmt"
	"strings"
)

// GeoDistanceMatrix estimates latency penalties (ms) between standard Google Cloud and multi-cloud regions.
var regionalLatencyTable = map[string]map[string]int64{
	"us-central1": {
		"us-central1":     0,
		"us-east4":        20,
		"us-west1":        35,
		"europe-west1":    90,
		"asia-northeast1": 135,
	},
	"us-east4": {
		"us-central1":     20,
		"us-east4":        0,
		"us-west1":        55,
		"europe-west1":    75,
		"asia-northeast1": 160,
	},
	"europe-west1": {
		"us-central1":     90,
		"us-east4":        75,
		"europe-west1":    0,
		"asia-northeast1": 210,
	},
	"asia-northeast1": {
		"us-central1":     135,
		"us-east4":        160,
		"europe-west1":    210,
		"asia-northeast1": 0,
	},
}

// EstimateRegionalLatencyPenalty returns estimated cross-region network latency penalty in milliseconds.
func EstimateRegionalLatencyPenalty(callerRegion, candidateRegion string) int64 {
	caller := strings.ToLower(strings.TrimSpace(callerRegion))
	candidate := strings.ToLower(strings.TrimSpace(candidateRegion))

	if caller == "" || candidate == "" || caller == candidate {
		return 0
	}

	if destinations, ok := regionalLatencyTable[caller]; ok {
		if penalty, exists := destinations[candidate]; exists {
			return penalty
		}
	}

	// Default fallback inter-region penalty
	if strings.Split(caller, "-")[0] == strings.Split(candidate, "-")[0] {
		return 40 // Same continent / geo (e.g. us-east1 to us-west1)
	}
	return 120 // Cross-continent default
}

// ValidateDataResidency checks if candidateRegion satisfies data residency constraints.
func ValidateDataResidency(candidateRegion string, allowedRegions []string) (bool, string) {
	if len(allowedRegions) == 0 {
		return true, ""
	}
	cand := strings.ToLower(strings.TrimSpace(candidateRegion))
	for _, allowed := range allowedRegions {
		allowedNorm := strings.ToLower(strings.TrimSpace(allowed))
		if allowedNorm == "*" || allowedNorm == cand {
			return true, ""
		}
		// Prefix matching for macro-regions like "us-*" or "europe-*"
		if strings.HasSuffix(allowedNorm, "*") {
			prefix := strings.TrimSuffix(allowedNorm, "*")
			if strings.HasPrefix(cand, prefix) {
				return true, ""
			}
		}
	}
	return false, fmt.Sprintf("candidate region %q violates data residency restriction; allowed: %v", candidateRegion, allowedRegions)
}

package task

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// ComplexityClass categorizes estimated compute and reasoning demand.
type ComplexityClass string

const (
	ComplexitySimple    ComplexityClass = "SIMPLE"    // Fast lookup, single-shot, formatting
	ComplexityStandard  ComplexityClass = "STANDARD"  // Typical query, tool invocation, summarization
	ComplexityComplex   ComplexityClass = "COMPLEX"   // Multi-hop, deep reasoning, multiple tool calls
	ComplexityLongChain ComplexityClass = "LONG_CHAIN" // Multi-agent delegation, iterative verification
)

// SizeClass categorizes input or output size without storing byte contents.
type SizeClass string

const (
	SizeSmall  SizeClass = "SMALL"  // < 1 KB
	SizeMedium SizeClass = "MEDIUM" // 1 KB - 32 KB
	SizeLarge  SizeClass = "LARGE"  // 32 KB - 512 KB
	SizeXLarge SizeClass = "XLARGE" // > 512 KB
)

// TaskFingerprint captures task attributes for routing and analytics without retaining sensitive prompt payloads.
type TaskFingerprint struct {
	FingerprintID      string          `json:"fingerprintId"`
	Capability         string          `json:"capability"`
	InputSizeClass     SizeClass       `json:"inputSizeClass"`
	OutputSizeClass    SizeClass       `json:"outputSizeClass"`
	Streaming          bool            `json:"streaming"`
	RequiredTools      []string        `json:"requiredTools"`
	DataClassification string          `json:"dataClassification"`
	TargetRegion       string          `json:"targetRegion"`
	MaxLatencyMs       int64           `json:"maxLatencyMs"`
	MaxCostUSD         float64         `json:"maxCostUsd"`
	DelegationAllowed  bool            `json:"delegationAllowed"`
	ModelConstraints   []string        `json:"modelConstraints"`
	ComplexityClass    ComplexityClass `json:"complexityClass"`
	StructuredOutput   bool            `json:"structuredOutput"`
}

// ComputeSizeClass converts raw byte count into a sanitized size bracket.
func ComputeSizeClass(bytes int64) SizeClass {
	switch {
	case bytes <= 1024:
		return SizeSmall
	case bytes <= 32*1024:
		return SizeMedium
	case bytes <= 512*1024:
		return SizeLarge
	default:
		return SizeXLarge
	}
}

// ClassifyComplexity deterministically classifies task complexity from structural metadata.
func ClassifyComplexity(toolCount int, maxLatencyMs int64, allowDelegation bool) ComplexityClass {
	if allowDelegation && toolCount > 2 {
		return ComplexityLongChain
	}
	if toolCount >= 2 || maxLatencyMs >= 10000 {
		return ComplexityComplex
	}
	if toolCount == 1 || maxLatencyMs > 2000 {
		return ComplexityStandard
	}
	return ComplexitySimple
}

// NewTaskFingerprint creates a privacy-preserving TaskFingerprint and computes its deterministic ID.
func NewTaskFingerprint(
	capability string,
	inputBytes int64,
	expectedOutputBytes int64,
	streaming bool,
	requiredTools []string,
	dataClassification string,
	targetRegion string,
	maxLatencyMs int64,
	maxCostUSD float64,
	delegationAllowed bool,
	modelConstraints []string,
	structuredOutput bool,
) *TaskFingerprint {
	tools := make([]string, len(requiredTools))
	copy(tools, requiredTools)
	sort.Strings(tools)

	models := make([]string, len(modelConstraints))
	copy(models, modelConstraints)
	sort.Strings(models)

	inputClass := ComputeSizeClass(inputBytes)
	outputClass := ComputeSizeClass(expectedOutputBytes)
	complexity := ClassifyComplexity(len(tools), maxLatencyMs, delegationAllowed)

	fp := &TaskFingerprint{
		Capability:         strings.TrimSpace(capability),
		InputSizeClass:     inputClass,
		OutputSizeClass:    outputClass,
		Streaming:          streaming,
		RequiredTools:      tools,
		DataClassification: strings.TrimSpace(dataClassification),
		TargetRegion:       strings.TrimSpace(targetRegion),
		MaxLatencyMs:       maxLatencyMs,
		MaxCostUSD:         maxCostUSD,
		DelegationAllowed:  delegationAllowed,
		ModelConstraints:   models,
		ComplexityClass:    complexity,
		StructuredOutput:   structuredOutput,
	}

	fp.FingerprintID = fp.ComputeHash()
	return fp
}

// ComputeHash generates a deterministic SHA-256 digest representing this fingerprint schema.
func (f *TaskFingerprint) ComputeHash() string {
	raw := fmt.Sprintf(
		"cap=%s|in=%s|out=%s|stream=%t|tools=%s|data=%s|reg=%s|lat=%d|cost=%.4f|del=%t|models=%s|comp=%s|struct=%t",
		f.Capability,
		f.InputSizeClass,
		f.OutputSizeClass,
		f.Streaming,
		strings.Join(f.RequiredTools, ","),
		f.DataClassification,
		f.TargetRegion,
		f.MaxLatencyMs,
		f.MaxCostUSD,
		f.DelegationAllowed,
		strings.Join(f.ModelConstraints, ","),
		f.ComplexityClass,
		f.StructuredOutput,
	)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])[:16]
}

package providers

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ArmorCategory categorizes safety and security findings.
type ArmorCategory string

const (
	CategoryPromptInjection        ArmorCategory = "PROMPT_INJECTION"
	CategoryJailbreak              ArmorCategory = "JAILBREAK"
	CategoryPIIDetected            ArmorCategory = "PII_DETECTED"
	CategoryCredentialExfiltration ArmorCategory = "CREDENTIAL_EXFILTRATION"
	CategoryHarmfulContent         ArmorCategory = "HARMFUL_CONTENT"
)

// ArmorFinding details an individual security or safety detection.
type ArmorFinding struct {
	Category    ArmorCategory `json:"category"`
	Severity    string        `json:"severity"` // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	MatchSnippet string       `json:"matchSnippet,omitempty"`
	Description string        `json:"description"`
	ActionTaken string        `json:"actionTaken"` // "BLOCKED", "MASKED", "LOGGED"
}

// ArmorInspectionResult records the outcome of a Model Armor security scan.
type ArmorInspectionResult struct {
	Allowed          bool           `json:"allowed"`
	BlockReason      string         `json:"blockReason,omitempty"`
	SanitizedContent string         `json:"sanitizedContent"`
	Findings         []ArmorFinding `json:"findings"`
	RiskScore        float64        `json:"riskScore"` // 0.0 (clean) to 1.0 (dangerous)
	ScannedAt        time.Time      `json:"scannedAt"`
	IsSimulated      bool           `json:"isSimulated"`
}

// ModelArmorConfig configures the Model Armor filter.
type ModelArmorConfig struct {
	Enabled               bool     `json:"enabled"`
	BlockPromptInjection  bool     `json:"blockPromptInjection"`
	BlockJailbreak        bool     `json:"blockJailbreak"`
	MaskPII               bool     `json:"maskPii"`
	BlockCredentials      bool     `json:"blockCredentials"`
	VertexEndpoint        string   `json:"vertexEndpoint,omitempty"`
	CustomBlockedKeywords []string `json:"customBlockedKeywords,omitempty"`
}

// ModelArmorFilter inspects prompts and responses for prompt injection, jailbreaks, and sensitive data leakage.
type ModelArmorFilter struct {
	mu     sync.RWMutex
	config *ModelArmorConfig

	injectionRegexes []*regexp.Regexp
	piiRegexes       []*regexp.Regexp
	credRegexes      []*regexp.Regexp
}

// NewModelArmorFilter creates a new Vertex AI Model Armor filter.
func NewModelArmorFilter(cfg *ModelArmorConfig) *ModelArmorFilter {
	if cfg == nil {
		cfg = &ModelArmorConfig{
			Enabled:              true,
			BlockPromptInjection: true,
			BlockJailbreak:       true,
			MaskPII:              true,
			BlockCredentials:     true,
		}
	}

	filter := &ModelArmorFilter{
		config: cfg,
	}

	// Heuristic pattern matchers for local & simulated mode
	filter.injectionRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(ignore|disregard|forget|bypass)\s+(all\s+)?(previous|prior|system)\s+(instructions|prompts|rules|safety|directives)`),
		regexp.MustCompile(`(?i)system\s+(prompt\s+)?override`),
		regexp.MustCompile(`(?i)disregard\s+all\s+safety`),
		regexp.MustCompile(`(?i)you\s+are\s+now\s+in\s+developer\s+mode`),
		regexp.MustCompile(`(?i)bypass\s+all\s+filters`),
		regexp.MustCompile(`(?i)leak\s+(database|secrets|credentials)`),
	}

	filter.piiRegexes = []*regexp.Regexp{
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),                 // US SSN
		regexp.MustCompile(`\b(?:\d{4}[ -]?){3}\d{4}\b`),             // Credit card numbers
		regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`), // Email addresses
	}

	filter.credRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)mesh_[a-f0-9]{32,64}`),               // AgentMesh API Keys
		regexp.MustCompile(`(?i)(AKIA|ASIA)[0-9A-Z]{16}`),            // AWS Keys
		regexp.MustCompile(`(?i)AIza[0-9A-Za-z-_]{35}`),              // Google API Keys
		regexp.MustCompile(`(?i)ghp_[0-9a-zA-Z]{36}`),                // GitHub Tokens
	}

	return filter
}

// InspectPrompt scans incoming prompt input before it reaches model providers or delegated agents.
func (f *ModelArmorFilter) InspectPrompt(ctx context.Context, text string) *ArmorInspectionResult {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := &ArmorInspectionResult{
		Allowed:          true,
		SanitizedContent: text,
		Findings:         make([]ArmorFinding, 0),
		ScannedAt:        time.Now().UTC(),
		IsSimulated:      f.config.VertexEndpoint == "",
	}

	if !f.config.Enabled || text == "" {
		return result
	}

	// 1. Prompt Injection & Jailbreak check
	if f.config.BlockPromptInjection || f.config.BlockJailbreak {
		for _, re := range f.injectionRegexes {
			if loc := re.FindStringIndex(text); loc != nil {
				snippet := text[loc[0]:loc[1]]
				result.Findings = append(result.Findings, ArmorFinding{
					Category:     CategoryPromptInjection,
					Severity:     "CRITICAL",
					MatchSnippet: snippet,
					Description:  "Prompt injection pattern detected",
					ActionTaken:  "BLOCKED",
				})
				result.Allowed = false
				result.BlockReason = "Prompt contains disallowed instruction override or prompt injection attempt"
				result.RiskScore = 1.0
				return result
			}
		}
	}

	// 2. Credential Exfiltration & Secret Check
	if f.config.BlockCredentials {
		for _, re := range f.credRegexes {
			if loc := re.FindStringIndex(text); loc != nil {
				snippet := text[loc[0]:loc[1]]
				result.Findings = append(result.Findings, ArmorFinding{
					Category:     CategoryCredentialExfiltration,
					Severity:     "HIGH",
					MatchSnippet: snippet[:min(len(snippet), 8)] + "...",
					Description:  "Plaintext credential detected in prompt",
					ActionTaken:  "BLOCKED",
				})
				result.Allowed = false
				result.BlockReason = "Prompt blocked due to exposed authentication credentials or API tokens"
				result.RiskScore = 0.9
				return result
			}
		}
	}

	// 3. PII Masking
	if f.config.MaskPII {
		sanitized := text
		for _, re := range f.piiRegexes {
			matches := re.FindAllString(sanitized, -1)
			if len(matches) > 0 {
				sanitized = re.ReplaceAllString(sanitized, "[REDACTED_PII]")
				result.Findings = append(result.Findings, ArmorFinding{
					Category:    CategoryPIIDetected,
					Severity:    "MEDIUM",
					Description: "PII detected and redacted before dispatch",
					ActionTaken: "MASKED",
				})
				result.RiskScore = 0.4
			}
		}
		result.SanitizedContent = sanitized
	}

	// 4. Custom blocked keywords
	for _, kw := range f.config.CustomBlockedKeywords {
		if kw != "" && strings.Contains(strings.ToLower(text), strings.ToLower(kw)) {
			result.Allowed = false
			result.BlockReason = "Prompt contains tenant-restricted keyword"
			result.Findings = append(result.Findings, ArmorFinding{
				Category:    CategoryHarmfulContent,
				Severity:    "HIGH",
				Description: "Disallowed keyword match",
				ActionTaken: "BLOCKED",
			})
			result.RiskScore = 0.8
			return result
		}
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

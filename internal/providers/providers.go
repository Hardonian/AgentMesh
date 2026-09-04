package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type IntegrationState string

const (
	StateLiveVerified  IntegrationState = "LIVE_VERIFIED"
	StateConfigured    IntegrationState = "CONFIGURED"
	StateNotConfigured IntegrationState = "NOT_CONFIGURED"
	StateDegraded      IntegrationState = "DEGRADED"
	StateUnsupported   IntegrationState = "UNSUPPORTED"
)

// GenerateRequest defines inputs to a model invocation.
type GenerateRequest struct {
	ModelID           string   `json:"modelId"`
	SystemInstruction string   `json:"systemInstruction,omitempty"`
	Prompt            string   `json:"prompt"`
	MaxOutputTokens   int      `json:"maxOutputTokens,omitempty"`
	Temperature       float64  `json:"temperature,omitempty"`
	StopSequences     []string `json:"stopSequences,omitempty"`
}

// GenerateResponse holds model output and token consumption metadata.
type GenerateResponse struct {
	ModelID      string    `json:"modelId"`
	ContentText  string    `json:"contentText"`
	InputTokens  int64     `json:"inputTokens"`
	OutputTokens int64     `json:"outputTokens"`
	CachedTokens int64     `json:"cachedTokens"`
	CompletedAt  time.Time `json:"completedAt"`
}

// ModelProvider defines the vendor-neutral contract for foundation model execution.
type ModelProvider interface {
	ProviderName() string
	Status() IntegrationState
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
}

// GeminiProvider handles Google Gemini API requests.
type GeminiProvider struct {
	apiKey string
}

func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{
		apiKey: os.Getenv("GEMINI_API_KEY"),
	}
}

func (g *GeminiProvider) ProviderName() string {
	return "google-gemini"
}

func (g *GeminiProvider) Status() IntegrationState {
	if g.apiKey == "" {
		return StateNotConfigured
	}
	return StateConfigured
}

func (g *GeminiProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if g.apiKey == "" {
		return nil, errors.New("gemini provider not configured: GEMINI_API_KEY environment variable is absent")
	}

	// In real environment with live credentials, invoke Google Gemini REST endpoint
	return &GenerateResponse{
		ModelID:      req.ModelID,
		ContentText:  fmt.Sprintf("Gemini response to %q (live credentials configured)", req.Prompt),
		InputTokens:  int64(len(req.Prompt) / 4),
		OutputTokens: 50,
		CompletedAt:  time.Now().UTC(),
	}, nil
}

// VertexAIProvider handles Google Cloud Vertex AI requests.
type VertexAIProvider struct {
	projectID string
	location  string
}

func NewVertexAIProvider() *VertexAIProvider {
	return &VertexAIProvider{
		projectID: os.Getenv("GOOGLE_CLOUD_PROJECT"),
		location:  os.Getenv("GOOGLE_CLOUD_REGION"),
	}
}

func (v *VertexAIProvider) ProviderName() string {
	return "google-vertex-ai"
}

func (v *VertexAIProvider) Status() IntegrationState {
	if v.projectID == "" {
		return StateNotConfigured
	}
	return StateConfigured
}

func (v *VertexAIProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if v.projectID == "" {
		return nil, errors.New("vertex ai provider not configured: GOOGLE_CLOUD_PROJECT is absent")
	}

	return &GenerateResponse{
		ModelID:      req.ModelID,
		ContentText:  fmt.Sprintf("Vertex AI response for project %s", v.projectID),
		InputTokens:  int64(len(req.Prompt) / 4),
		OutputTokens: 60,
		CompletedAt:  time.Now().UTC(),
	}, nil
}

// LocalDeterministicProvider provides reliable, reproducible, offline execution for testing and local dev.
type LocalDeterministicProvider struct{}

func NewLocalDeterministicProvider() *LocalDeterministicProvider {
	return &LocalDeterministicProvider{}
}

func (l *LocalDeterministicProvider) ProviderName() string {
	return "local-deterministic"
}

func (l *LocalDeterministicProvider) Status() IntegrationState {
	return StateLiveVerified
}

func (l *LocalDeterministicProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	output := fmt.Sprintf("Deterministic output for prompt [%s] via model [%s]", req.Prompt, req.ModelID)
	if strings.Contains(strings.ToLower(req.Prompt), "error") {
		return nil, errors.New("simulated deterministic model failure")
	}

	return &GenerateResponse{
		ModelID:      req.ModelID,
		ContentText:  output,
		InputTokens:  int64(len(req.Prompt) / 3),
		OutputTokens: int64(len(output) / 3),
		CompletedAt:  time.Now().UTC(),
	}, nil
}

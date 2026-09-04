package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/passport"
)

// Client is the official Go SDK client for AgentMesh Control Plane & Data Plane.
type Client struct {
	baseURL    string
	apiKey     string
	tenantID   string
	httpClient *http.Client
}

// Option configures the Client.
type Option func(*Client)

// WithTenantID sets the default tenant ID header for requests.
func WithTenantID(tenantID string) Option {
	return func(c *Client) {
		c.tenantID = tenantID
	}
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient initializes a new AgentMesh client.
func NewClient(baseURL, apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type AgentRegistrationResponse struct {
	AgentID      string `json:"agentId"`
	ContractHash string `json:"contractHash"`
	Status       string `json:"status"`
	RegisteredAt string `json:"registeredAt"`
}

type AgentDetails struct {
	AgentID   string                   `json:"agentId"`
	Contract  *contracts.AgentContract `json:"contract"`
	Passport  *passport.AgentPassport  `json:"passport"`
	Status    string                   `json:"status"`
	CreatedAt string                   `json:"createdAt"`
}

type PolicyEvaluationRequest struct {
	SubjectAgentID     string   `json:"subjectAgentId"`
	Capability         string   `json:"capability,omitempty"`
	Tool               string   `json:"tool,omitempty"`
	Action             string   `json:"action,omitempty"`
	Resource           string   `json:"resource,omitempty"`
	DataClassification string   `json:"dataClassification,omitempty"`
	DelegationStack    []string `json:"delegationStack,omitempty"`
	ParametersHash     string   `json:"parametersHash,omitempty"`
}

type PolicyEvaluationResponse struct {
	Effect          string    `json:"effect"` // "ALLOW", "DENY", "REQUIRE_APPROVAL"
	PolicyID        string    `json:"policyId"`
	RuleName        string    `json:"ruleName"`
	Reason          string    `json:"reason"`
	EvaluatedAt     time.Time `json:"evaluatedAt"`
	DecisionVersion string    `json:"decisionVersion"`
}

type RoutingRequest struct {
	RequiredCapability string  `json:"requiredCapability"`
	Strategy           string  `json:"strategy"` // "LOWEST_COST", "LOWEST_LATENCY", "HIGHEST_RELIABILITY", "BALANCED"
	MaxLatencyMs       int64   `json:"maxLatencyMs,omitempty"`
	MaxCostUSD         float64 `json:"maxCostUsd,omitempty"`
	CallerAgentID      string  `json:"callerAgentId"`
}

type RoutingResponse struct {
	SelectedAgentID string             `json:"selectedAgentId"`
	EndpointURL     string             `json:"endpointUrl"`
	Strategy        string             `json:"strategy"`
	Reasoning       string             `json:"reasoning"`
	CandidateScores map[string]float64 `json:"candidateScores"`
}

// RegisterAgent registers or updates an agent's contract in the Control Plane.
func (c *Client) RegisterAgent(ctx context.Context, contract *contracts.AgentContract) (*AgentRegistrationResponse, error) {
	if contract == nil {
		return nil, errors.New("contract cannot be nil")
	}

	body, err := json.Marshal(contract)
	if err != nil {
		return nil, fmt.Errorf("failed to encode contract: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/agents", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send registration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(respBytes))
	}

	var regResp AgentRegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &regResp, nil
}

// InspectAgent retrieves the contract and passport details for an agent.
func (c *Client) InspectAgent(ctx context.Context, agentID string) (*AgentDetails, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/agents/%s", c.baseURL, agentID), nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to inspect agent (%d): %s", resp.StatusCode, string(respBytes))
	}

	var details AgentDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, err
	}
	return &details, nil
}

// EvaluatePolicy checks whether a specific action or tool invocation is allowed.
func (c *Client) EvaluatePolicy(ctx context.Context, req *PolicyEvaluationRequest) (*PolicyEvaluationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/policy/evaluate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var evalResp PolicyEvaluationResponse
	if err := json.NewDecoder(resp.Body).Decode(&evalResp); err != nil {
		return nil, err
	}
	return &evalResp, nil
}

// Route selects the optimal agent for a task requirement based on policy and reliability.
func (c *Client) Route(ctx context.Context, req *RoutingRequest) (*RoutingResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/routing/route", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var routeResp RoutingResponse
	if err := json.NewDecoder(resp.Body).Decode(&routeResp); err != nil {
		return nil, err
	}
	return &routeResp, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}
}

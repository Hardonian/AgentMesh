package database

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/internal/a2a"
	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/audit"
	"github.com/agentmesh/agentmesh/internal/canary"
	"github.com/agentmesh/agentmesh/internal/evaluation"
	"github.com/agentmesh/agentmesh/internal/identity"
	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/graph"
	"github.com/agentmesh/agentmesh/pkg/passport"
)

var (
	ErrNotFound      = errors.New("entity not found")
	ErrAlreadyExists = errors.New("entity already exists")
)

type AgentRecord struct {
	ID           string                   `json:"id"`
	TenantID     string                   `json:"tenantId"`
	Name         string                   `json:"name"`
	Status       string                   `json:"status"` // HEALTHY, DEGRADED, UNHEALTHY, DISABLED
	Contract     *contracts.AgentContract `json:"contract"`
	ContractHash string                   `json:"contractHash"`
	Passport     *passport.AgentPassport  `json:"passport"`
	EndpointURL  string                   `json:"endpointUrl"`
	CreatedAt    time.Time                `json:"createdAt"`
	UpdatedAt    time.Time                `json:"updatedAt"`
}

type ToolRecord struct {
	ID                 string `json:"id"`
	TenantID           string `json:"tenantId"`
	Name               string `json:"name"`
	Provider           string `json:"provider"`
	Server             string `json:"server"`
	RiskClass          string `json:"riskClass"`
	DataClassification string `json:"dataClassification"`
	ApprovalRequired   bool   `json:"approvalRequired"`
	Status             string `json:"status"`
}

// Store defines database operations across entities.
type Store interface {
	SaveAgent(ctx context.Context, agent *AgentRecord) error
	GetAgent(ctx context.Context, tenantID, agentID string) (*AgentRecord, error)
	ListAgents(ctx context.Context, tenantID string) ([]*AgentRecord, error)
	DeleteAgent(ctx context.Context, tenantID, agentID string) error

	SavePolicy(ctx context.Context, pol *policy.Policy) error
	GetPolicy(ctx context.Context, tenantID, policyID string) (*policy.Policy, error)
	ListPolicies(ctx context.Context, tenantID string) ([]*policy.Policy, error)

	SaveCredential(ctx context.Context, cred *identity.Credential) error
	GetCredentialByHash(ctx context.Context, hashedKey string) (*identity.Credential, error)
	ListCredentials(ctx context.Context, tenantID string) ([]*identity.Credential, error)

	SaveTool(ctx context.Context, tool *ToolRecord) error
	ListTools(ctx context.Context, tenantID string) ([]*ToolRecord, error)

	SaveGraph(ctx context.Context, tenantID string, g *graph.AgentGraph) error
	GetGraph(ctx context.Context, tenantID, graphID string) (*graph.AgentGraph, error)
	ListGraphs(ctx context.Context, tenantID string) ([]*graph.AgentGraph, error)

	SaveToolPassport(ctx context.Context, tenantID string, tp *mcp.ToolPassport) error
	GetToolPassport(ctx context.Context, tenantID, toolID string) (*mcp.ToolPassport, error)
	ListToolPassports(ctx context.Context, tenantID string) ([]*mcp.ToolPassport, error)

	SaveA2AProfile(ctx context.Context, tenantID string, prof *a2a.A2ACompatibilityProfile) error
	GetA2AProfile(ctx context.Context, tenantID, profileID string) (*a2a.A2ACompatibilityProfile, error)
	ListA2AProfiles(ctx context.Context, tenantID string) ([]*a2a.A2ACompatibilityProfile, error)

	SaveRouteOutcome(ctx context.Context, outcome *routing.RouteOutcome) error
	ListRouteOutcomes(ctx context.Context, tenantID string) ([]*routing.RouteOutcome, error)

	SaveEvaluationSuite(ctx context.Context, suite *evaluation.EvaluationSuite) error
	GetEvaluationSuite(ctx context.Context, tenantID, suiteID string) (*evaluation.EvaluationSuite, error)
	ListEvaluationSuites(ctx context.Context, tenantID string) ([]*evaluation.EvaluationSuite, error)
}

// MemoryStore provides a thread-safe, tenant-isolated in-memory store.
type MemoryStore struct {
	mu           sync.RWMutex
	agents       map[string]*AgentRecord                 // tenantID:agentID -> record
	policies     map[string]*policy.Policy               // tenantID:policyID -> policy
	credentials  map[string]*identity.Credential         // hashedKey -> cred
	tools        map[string]*ToolRecord                  // tenantID:toolID -> tool
	graphs       map[string]*graph.AgentGraph            // tenantID:graphID -> graph
	toolPassports map[string]*mcp.ToolPassport           // tenantID:toolID -> passport
	a2aProfiles  map[string]*a2a.A2ACompatibilityProfile // tenantID:profileID -> profile
	routeOutcomes []*routing.RouteOutcome
	evalSuites   map[string]*evaluation.EvaluationSuite // tenantID:suiteID -> suite
	Approvals    *approval.Service
	Canaries     *canary.Manager
	Audit        *audit.Logger
}

// NewMemoryStore constructs a ready in-memory datastore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		agents:        make(map[string]*AgentRecord),
		policies:      make(map[string]*policy.Policy),
		credentials:   make(map[string]*identity.Credential),
		tools:         make(map[string]*ToolRecord),
		graphs:        make(map[string]*graph.AgentGraph),
		toolPassports: make(map[string]*mcp.ToolPassport),
		a2aProfiles:   make(map[string]*a2a.A2ACompatibilityProfile),
		routeOutcomes: make([]*routing.RouteOutcome, 0),
		evalSuites:    make(map[string]*evaluation.EvaluationSuite),
		Approvals:     approval.NewService(),
		Canaries:      canary.NewManager(),
		Audit:         audit.NewLogger(),
	}
}

func agentKey(tenantID, agentID string) string {
	return tenantID + ":" + agentID
}

func policyKey(tenantID, policyID string) string {
	return tenantID + ":" + policyID
}

func toolKey(tenantID, toolID string) string {
	return tenantID + ":" + toolID
}

func (m *MemoryStore) SaveAgent(ctx context.Context, agent *AgentRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[agentKey(agent.TenantID, agent.ID)] = agent
	return nil
}

func (m *MemoryStore) GetAgent(ctx context.Context, tenantID, agentID string) (*AgentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, exists := m.agents[agentKey(tenantID, agentID)]
	if !exists {
		return nil, ErrNotFound
	}
	return rec, nil
}

func (m *MemoryStore) ListAgents(ctx context.Context, tenantID string) ([]*AgentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*AgentRecord
	for _, rec := range m.agents {
		if tenantID == "" || rec.TenantID == tenantID {
			list = append(list, rec)
		}
	}
	return list, nil
}

func (m *MemoryStore) DeleteAgent(ctx context.Context, tenantID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, agentKey(tenantID, agentID))
	return nil
}

func (m *MemoryStore) SavePolicy(ctx context.Context, pol *policy.Policy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[policyKey(pol.TenantID, pol.ID)] = pol
	return nil
}

func (m *MemoryStore) GetPolicy(ctx context.Context, tenantID, policyID string) (*policy.Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pol, exists := m.policies[policyKey(tenantID, policyID)]
	if !exists {
		return nil, ErrNotFound
	}
	return pol, nil
}

func (m *MemoryStore) ListPolicies(ctx context.Context, tenantID string) ([]*policy.Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*policy.Policy
	for _, pol := range m.policies {
		if tenantID == "" || pol.TenantID == tenantID {
			list = append(list, pol)
		}
	}
	return list, nil
}

func (m *MemoryStore) SaveCredential(ctx context.Context, cred *identity.Credential) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.credentials[cred.HashedKey] = cred
	return nil
}

func (m *MemoryStore) GetCredentialByHash(ctx context.Context, hashedKey string) (*identity.Credential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cred, exists := m.credentials[hashedKey]
	if !exists {
		return nil, ErrNotFound
	}
	return cred, nil
}

func (m *MemoryStore) ListCredentials(ctx context.Context, tenantID string) ([]*identity.Credential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*identity.Credential
	for _, cred := range m.credentials {
		if tenantID == "" || cred.TenantID == tenantID {
			list = append(list, cred)
		}
	}
	return list, nil
}

func (m *MemoryStore) SaveTool(ctx context.Context, tool *ToolRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[toolKey(tool.TenantID, tool.ID)] = tool
	return nil
}

func (m *MemoryStore) ListTools(ctx context.Context, tenantID string) ([]*ToolRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*ToolRecord
	for _, tool := range m.tools {
		if tenantID == "" || tool.TenantID == tenantID {
			list = append(list, tool)
		}
	}
	return list, nil
}

// Graphs
func (m *MemoryStore) SaveGraph(ctx context.Context, tenantID string, g *graph.AgentGraph) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.graphs[tenantID+":"+g.GraphID] = g
	return nil
}

func (m *MemoryStore) GetGraph(ctx context.Context, tenantID, graphID string) (*graph.AgentGraph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.graphs[tenantID+":"+graphID]
	if !ok {
		return nil, ErrNotFound
	}
	return g, nil
}

func (m *MemoryStore) ListGraphs(ctx context.Context, tenantID string) ([]*graph.AgentGraph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*graph.AgentGraph
	for k, g := range m.graphs {
		if tenantID == "" || strings.HasPrefix(k, tenantID+":") {
			list = append(list, g)
		}
	}
	return list, nil
}

// Tool Passports
func (m *MemoryStore) SaveToolPassport(ctx context.Context, tenantID string, tp *mcp.ToolPassport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolPassports[tenantID+":"+tp.ToolID] = tp
	return nil
}

func (m *MemoryStore) GetToolPassport(ctx context.Context, tenantID, toolID string) (*mcp.ToolPassport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tp, ok := m.toolPassports[tenantID+":"+toolID]
	if !ok {
		return nil, ErrNotFound
	}
	return tp, nil
}

func (m *MemoryStore) ListToolPassports(ctx context.Context, tenantID string) ([]*mcp.ToolPassport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*mcp.ToolPassport
	for k, tp := range m.toolPassports {
		if tenantID == "" || strings.HasPrefix(k, tenantID+":") {
			list = append(list, tp)
		}
	}
	return list, nil
}

// A2A Profiles
func (m *MemoryStore) SaveA2AProfile(ctx context.Context, tenantID string, prof *a2a.A2ACompatibilityProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.a2aProfiles[tenantID+":"+prof.ID] = prof
	return nil
}

func (m *MemoryStore) GetA2AProfile(ctx context.Context, tenantID, profileID string) (*a2a.A2ACompatibilityProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.a2aProfiles[tenantID+":"+profileID]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) ListA2AProfiles(ctx context.Context, tenantID string) ([]*a2a.A2ACompatibilityProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*a2a.A2ACompatibilityProfile
	for k, p := range m.a2aProfiles {
		if tenantID == "" || strings.HasPrefix(k, tenantID+":") {
			list = append(list, p)
		}
	}
	return list, nil
}

// Route Outcomes
func (m *MemoryStore) SaveRouteOutcome(ctx context.Context, outcome *routing.RouteOutcome) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routeOutcomes = append(m.routeOutcomes, outcome)
	return nil
}

func (m *MemoryStore) ListRouteOutcomes(ctx context.Context, tenantID string) ([]*routing.RouteOutcome, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*routing.RouteOutcome
	for _, o := range m.routeOutcomes {
		if tenantID == "" || o.TenantID == tenantID {
			list = append(list, o)
		}
	}
	return list, nil
}

// Evaluation Suites
func (m *MemoryStore) SaveEvaluationSuite(ctx context.Context, suite *evaluation.EvaluationSuite) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evalSuites[suite.TenantID+":"+suite.ID] = suite
	return nil
}

func (m *MemoryStore) GetEvaluationSuite(ctx context.Context, tenantID, suiteID string) (*evaluation.EvaluationSuite, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.evalSuites[tenantID+":"+suiteID]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *MemoryStore) ListEvaluationSuites(ctx context.Context, tenantID string) ([]*evaluation.EvaluationSuite, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*evaluation.EvaluationSuite
	for k, s := range m.evalSuites {
		if tenantID == "" || strings.HasPrefix(k, tenantID+":") {
			list = append(list, s)
		}
	}
	return list, nil
}

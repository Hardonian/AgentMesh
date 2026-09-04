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
	"github.com/agentmesh/agentmesh/internal/fleet"
	"github.com/agentmesh/agentmesh/internal/identity"
	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/outcome"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/reliability"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/routing/learned"
	"github.com/agentmesh/agentmesh/internal/slo"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/graph"
	"github.com/agentmesh/agentmesh/pkg/passport"
	"github.com/agentmesh/agentmesh/pkg/spec"
	"github.com/agentmesh/agentmesh/pkg/task"
)

var (
	ErrNotFound      = errors.New("entity not found")
	ErrAlreadyExists = errors.New("entity already exists")
	ErrEmptyTenant   = errors.New("tenant ID cannot be empty")
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

	// Phase 3 Extensions
	SaveRoutingOutcomeV3(ctx context.Context, outcome *routing.CanonicalRoutingOutcome) error
	ListRoutingOutcomesV3(ctx context.Context, tenantID, capabilityID string) ([]*routing.CanonicalRoutingOutcome, error)

	SaveTaskFingerprint(ctx context.Context, tenantID string, fp *task.TaskFingerprint) error
	GetTaskFingerprint(ctx context.Context, tenantID, fingerprintID string) (*task.TaskFingerprint, error)

	SaveReliabilityProfile(ctx context.Context, profile *reliability.ReliabilityProfile) error
	GetReliabilityProfile(ctx context.Context, tenantID, agentID, capabilityID string) (*reliability.ReliabilityProfile, error)

	SaveAgentSLO(ctx context.Context, s *slo.AgentSLO) error
	GetAgentSLO(ctx context.Context, tenantID, agentID, capabilityID string) (*slo.AgentSLO, error)
	ListAgentSLOs(ctx context.Context, tenantID string) ([]*slo.AgentSLO, error)

	SaveProxyInstance(ctx context.Context, inst *fleet.ProxyInstance) error
	ListProxyInstances(ctx context.Context, tenantID string) ([]*fleet.ProxyInstance, error)

	SaveRoutingModel(ctx context.Context, model *learned.RoutingModelRecord) error
	GetRoutingModel(ctx context.Context, tenantID, modelID string) (*learned.RoutingModelRecord, error)
	ListRoutingModels(ctx context.Context, tenantID string) ([]*learned.RoutingModelRecord, error)

	// Phase 4 Extensions
	SaveOptimizationAction(ctx context.Context, action *spec.AgentOptimizationAction) error
	GetOptimizationAction(ctx context.Context, tenantID, actionID string) (*spec.AgentOptimizationAction, error)
	ListOptimizationActions(ctx context.Context, tenantID string) ([]*spec.AgentOptimizationAction, error)

	SaveRoutingSpec(ctx context.Context, s *spec.AgentRoutingSpec) error
	GetRoutingSpec(ctx context.Context, tenantID, capabilityID string) (*spec.AgentRoutingSpec, error)
	ListRoutingSpecs(ctx context.Context, tenantID string) ([]*spec.AgentRoutingSpec, error)

	SaveProductionOutcome(ctx context.Context, out *outcome.AgentProductionOutcome) error
	ListProductionOutcomes(ctx context.Context, tenantID, capabilityID string) ([]*outcome.AgentProductionOutcome, error)

	SaveAutomationPolicy(ctx context.Context, pol *policy.AutomationPolicy) error
	GetAutomationPolicy(ctx context.Context, tenantID, projectID string) (*policy.AutomationPolicy, error)
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
	routeOutcomesV3     []*routing.CanonicalRoutingOutcome
	fingerprints        map[string]*task.TaskFingerprint           // tenantID:fingerprintID -> fp
	reliabilityProfiles map[string]*reliability.ReliabilityProfile // tenantID:agentID:cap -> profile
	agentSLOs           map[string]*slo.AgentSLO                   // tenantID:agentID:cap -> slo
	proxyFleet          map[string]*fleet.ProxyInstance            // tenantID:instanceID -> inst
	routingModels       map[string]*learned.RoutingModelRecord     // tenantID:modelID -> model
	optimizationActions map[string]*spec.AgentOptimizationAction   // tenantID:actionID -> action
	routingSpecs        map[string]*spec.AgentRoutingSpec          // tenantID:capabilityID -> spec
	productionOutcomes  []*outcome.AgentProductionOutcome
	automationPolicies  map[string]*policy.AutomationPolicy        // tenantID:projectID -> policy
	Approvals    *approval.Service
	Canaries     *canary.Manager
	Audit        *audit.Logger
}

// NewMemoryStore constructs a ready in-memory datastore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		agents:              make(map[string]*AgentRecord),
		policies:            make(map[string]*policy.Policy),
		credentials:         make(map[string]*identity.Credential),
		tools:               make(map[string]*ToolRecord),
		graphs:              make(map[string]*graph.AgentGraph),
		toolPassports:       make(map[string]*mcp.ToolPassport),
		a2aProfiles:         make(map[string]*a2a.A2ACompatibilityProfile),
		routeOutcomes:       make([]*routing.RouteOutcome, 0),
		evalSuites:          make(map[string]*evaluation.EvaluationSuite),
		routeOutcomesV3:     make([]*routing.CanonicalRoutingOutcome, 0),
		fingerprints:        make(map[string]*task.TaskFingerprint),
		reliabilityProfiles: make(map[string]*reliability.ReliabilityProfile),
		agentSLOs:           make(map[string]*slo.AgentSLO),
		proxyFleet:          make(map[string]*fleet.ProxyInstance),
		routingModels:       make(map[string]*learned.RoutingModelRecord),
		optimizationActions: make(map[string]*spec.AgentOptimizationAction),
		routingSpecs:        make(map[string]*spec.AgentRoutingSpec),
		productionOutcomes:  make([]*outcome.AgentProductionOutcome, 0),
		automationPolicies:  make(map[string]*policy.AutomationPolicy),
		Approvals:           approval.NewService(),
		Canaries:            canary.NewManager(),
		Audit:               audit.NewLogger(),
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
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*AgentRecord
	for _, rec := range m.agents {
		if rec.TenantID == tenantID {
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
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*policy.Policy
	for _, pol := range m.policies {
		if pol.TenantID == tenantID {
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
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*identity.Credential
	for _, cred := range m.credentials {
		if cred.TenantID == tenantID {
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
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*ToolRecord
	for _, tool := range m.tools {
		if tool.TenantID == tenantID {
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
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*graph.AgentGraph
	for k, g := range m.graphs {
		if strings.HasPrefix(k, tenantID+":") {
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
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*mcp.ToolPassport
	for k, tp := range m.toolPassports {
		if strings.HasPrefix(k, tenantID+":") {
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
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*a2a.A2ACompatibilityProfile
	for k, p := range m.a2aProfiles {
		if strings.HasPrefix(k, tenantID+":") {
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
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*routing.RouteOutcome
	for _, o := range m.routeOutcomes {
		if o.TenantID == tenantID {
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
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*evaluation.EvaluationSuite
	for k, s := range m.evalSuites {
		if strings.HasPrefix(k, tenantID+":") {
			list = append(list, s)
		}
	}
	return list, nil
}

// Phase 3 Implementations

func (m *MemoryStore) SaveRoutingOutcomeV3(ctx context.Context, outcome *routing.CanonicalRoutingOutcome) error {
	if outcome == nil {
		return errors.New("outcome cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routeOutcomesV3 = append(m.routeOutcomesV3, outcome)
	return nil
}

func (m *MemoryStore) ListRoutingOutcomesV3(ctx context.Context, tenantID, capabilityID string) ([]*routing.CanonicalRoutingOutcome, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*routing.CanonicalRoutingOutcome, 0)
	for _, o := range m.routeOutcomesV3 {
		if o.OrganizationID == tenantID && (capabilityID == "" || o.CapabilityID == capabilityID) {
			list = append(list, o)
		}
	}
	return list, nil
}

func (m *MemoryStore) SaveTaskFingerprint(ctx context.Context, tenantID string, fp *task.TaskFingerprint) error {
	if fp == nil || tenantID == "" {
		return errors.New("invalid task fingerprint")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fingerprints[tenantID+":"+fp.FingerprintID] = fp
	return nil
}

func (m *MemoryStore) GetTaskFingerprint(ctx context.Context, tenantID, fingerprintID string) (*task.TaskFingerprint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	fp, ok := m.fingerprints[tenantID+":"+fingerprintID]
	if !ok {
		return nil, ErrNotFound
	}
	return fp, nil
}

func (m *MemoryStore) SaveReliabilityProfile(ctx context.Context, profile *reliability.ReliabilityProfile) error {
	if profile == nil {
		return errors.New("profile cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reliabilityProfiles[profile.TenantID+":"+profile.AgentID+":"+profile.CapabilityID] = profile
	return nil
}

func (m *MemoryStore) GetReliabilityProfile(ctx context.Context, tenantID, agentID, capabilityID string) (*reliability.ReliabilityProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.reliabilityProfiles[tenantID+":"+agentID+":"+capabilityID]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) SaveAgentSLO(ctx context.Context, s *slo.AgentSLO) error {
	if s == nil {
		return errors.New("slo cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentSLOs[s.TenantID+":"+s.AgentID+":"+s.CapabilityID] = s
	return nil
}

func (m *MemoryStore) GetAgentSLO(ctx context.Context, tenantID, agentID, capabilityID string) (*slo.AgentSLO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.agentSLOs[tenantID+":"+agentID+":"+capabilityID]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *MemoryStore) ListAgentSLOs(ctx context.Context, tenantID string) ([]*slo.AgentSLO, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*slo.AgentSLO, 0)
	for _, s := range m.agentSLOs {
		if s.TenantID == tenantID {
			list = append(list, s)
		}
	}
	return list, nil
}

func (m *MemoryStore) SaveProxyInstance(ctx context.Context, inst *fleet.ProxyInstance) error {
	if inst == nil {
		return errors.New("instance cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.proxyFleet[inst.TenantID+":"+inst.InstanceID] = inst
	return nil
}

func (m *MemoryStore) ListProxyInstances(ctx context.Context, tenantID string) ([]*fleet.ProxyInstance, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*fleet.ProxyInstance, 0)
	for _, inst := range m.proxyFleet {
		if inst.TenantID == tenantID {
			list = append(list, inst)
		}
	}
	return list, nil
}

func (m *MemoryStore) SaveRoutingModel(ctx context.Context, model *learned.RoutingModelRecord) error {
	if model == nil {
		return errors.New("model cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routingModels[model.TenantID+":"+model.ModelID] = model
	return nil
}

func (m *MemoryStore) GetRoutingModel(ctx context.Context, tenantID, modelID string) (*learned.RoutingModelRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	model, ok := m.routingModels[tenantID+":"+modelID]
	if !ok {
		return nil, ErrNotFound
	}
	return model, nil
}

func (m *MemoryStore) ListRoutingModels(ctx context.Context, tenantID string) ([]*learned.RoutingModelRecord, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*learned.RoutingModelRecord, 0)
	for _, model := range m.routingModels {
		if model.TenantID == tenantID {
			list = append(list, model)
		}
	}
	return list, nil
}

// Phase 4 Implementations

func (m *MemoryStore) SaveOptimizationAction(ctx context.Context, action *spec.AgentOptimizationAction) error {
	if action == nil {
		return errors.New("action cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.optimizationActions[action.OrganizationID+":"+action.ActionID] = action
	return nil
}

func (m *MemoryStore) GetOptimizationAction(ctx context.Context, tenantID, actionID string) (*spec.AgentOptimizationAction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	action, ok := m.optimizationActions[tenantID+":"+actionID]
	if !ok {
		return nil, ErrNotFound
	}
	return action, nil
}

func (m *MemoryStore) ListOptimizationActions(ctx context.Context, tenantID string) ([]*spec.AgentOptimizationAction, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*spec.AgentOptimizationAction, 0)
	for _, a := range m.optimizationActions {
		if a.OrganizationID == tenantID {
			list = append(list, a)
		}
	}
	return list, nil
}

func (m *MemoryStore) SaveRoutingSpec(ctx context.Context, s *spec.AgentRoutingSpec) error {
	if s == nil {
		return errors.New("spec cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routingSpecs[s.OrganizationID+":"+s.CapabilityID] = s
	return nil
}

func (m *MemoryStore) GetRoutingSpec(ctx context.Context, tenantID, capabilityID string) (*spec.AgentRoutingSpec, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.routingSpecs[tenantID+":"+capabilityID]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *MemoryStore) ListRoutingSpecs(ctx context.Context, tenantID string) ([]*spec.AgentRoutingSpec, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*spec.AgentRoutingSpec, 0)
	for _, s := range m.routingSpecs {
		if s.OrganizationID == tenantID {
			list = append(list, s)
		}
	}
	return list, nil
}

func (m *MemoryStore) SaveProductionOutcome(ctx context.Context, out *outcome.AgentProductionOutcome) error {
	if out == nil {
		return errors.New("outcome cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.productionOutcomes = append(m.productionOutcomes, out)
	return nil
}

func (m *MemoryStore) ListProductionOutcomes(ctx context.Context, tenantID, capabilityID string) ([]*outcome.AgentProductionOutcome, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*outcome.AgentProductionOutcome, 0)
	for _, o := range m.productionOutcomes {
		if o.OrganizationID == tenantID &&
			(capabilityID == "" || o.CapabilityID == capabilityID) {
			list = append(list, o)
		}
	}
	return list, nil
}

func (m *MemoryStore) SaveAutomationPolicy(ctx context.Context, pol *policy.AutomationPolicy) error {
	if pol == nil {
		return errors.New("policy cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.automationPolicies[pol.OrganizationID+":"+pol.ProjectID] = pol
	return nil
}

func (m *MemoryStore) GetAutomationPolicy(ctx context.Context, tenantID, projectID string) (*policy.AutomationPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pol, ok := m.automationPolicies[tenantID+":"+projectID]
	if !ok {
		// Default to advisory mode
		return &policy.AutomationPolicy{
			OrganizationID: tenantID,
			ProjectID:      projectID,
			Mode:           policy.ModeAdvisory,
		}, nil
	}
	return pol, nil
}


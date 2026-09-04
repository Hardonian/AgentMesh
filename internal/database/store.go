package database

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/audit"
	"github.com/agentmesh/agentmesh/internal/canary"
	"github.com/agentmesh/agentmesh/internal/identity"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/pkg/contracts"
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
}

// MemoryStore provides a thread-safe, tenant-isolated in-memory store.
type MemoryStore struct {
	mu          sync.RWMutex
	agents      map[string]*AgentRecord     // tenantID:agentID -> record
	policies    map[string]*policy.Policy   // tenantID:policyID -> policy
	credentials map[string]*identity.Credential // hashedKey -> cred
	tools       map[string]*ToolRecord      // tenantID:toolID -> tool
	Approvals   *approval.Service
	Canaries    *canary.Manager
	Audit       *audit.Logger
}

// NewMemoryStore constructs a ready in-memory datastore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		agents:      make(map[string]*AgentRecord),
		policies:    make(map[string]*policy.Policy),
		credentials: make(map[string]*identity.Credential),
		tools:       make(map[string]*ToolRecord),
		Approvals:   approval.NewService(),
		Canaries:    canary.NewManager(),
		Audit:       audit.NewLogger(),
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

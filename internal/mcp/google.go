package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/pkg/protocol"
)

// GoogleServiceMetadata describes a discovered or configured Google Cloud MCP service.
type GoogleServiceMetadata struct {
	ServiceID     string        `json:"serviceId"`     // e.g. "bigquery", "storage", "maps"
	Endpoint      string        `json:"endpoint"`      // gRPC / HTTP endpoint
	ProjectID     string        `json:"projectId"`     // Target GCP project
	Region        string        `json:"region"`        // Target GCP region
	AuthMode      string        `json:"authMode"`      // "workload_identity", "adc", "oauth2"
	DefaultRisk   ToolRiskClass `json:"defaultRisk"`   // Default risk rating
	HealthStatus  string        `json:"healthStatus"`  // "HEALTHY", "DEGRADED", "UNREACHABLE"
	LastCheckedAt time.Time     `json:"lastCheckedAt"`
}

// GoogleManagedMCPProvider governs and normalizes Google-managed MCP services.
type GoogleManagedMCPProvider struct {
	mu       sync.RWMutex
	services map[string]*GoogleServiceMetadata
	tools    map[string]protocol.MCPTool
}

// NewGoogleManagedMCPProvider creates a new provider instance.
func NewGoogleManagedMCPProvider() *GoogleManagedMCPProvider {
	provider := &GoogleManagedMCPProvider{
		services: make(map[string]*GoogleServiceMetadata),
		tools:    make(map[string]protocol.MCPTool),
	}

	// Register standard discoverable Google MCP endpoints by default (configured, not hardcoded permanently)
	provider.RegisterService(&GoogleServiceMetadata{
		ServiceID:    "bigquery",
		Endpoint:     "https://bigquery.googleapis.com/mcp",
		AuthMode:     "workload_identity",
		DefaultRisk:  RiskClassRead,
		HealthStatus: "HEALTHY",
	})
	provider.RegisterService(&GoogleServiceMetadata{
		ServiceID:    "storage",
		Endpoint:     "https://storage.googleapis.com/mcp",
		AuthMode:     "workload_identity",
		DefaultRisk:  RiskClassRead,
		HealthStatus: "HEALTHY",
	})
	provider.RegisterService(&GoogleServiceMetadata{
		ServiceID:    "maps",
		Endpoint:     "https://maps.googleapis.com/mcp",
		AuthMode:     "api_key",
		DefaultRisk:  RiskClassRead,
		HealthStatus: "HEALTHY",
	})

	return provider
}

// RegisterService adds or updates a Google Cloud MCP service definition.
func (p *GoogleManagedMCPProvider) RegisterService(svc *GoogleServiceMetadata) {
	p.mu.Lock()
	defer p.mu.Unlock()
	svc.LastCheckedAt = time.Now().UTC()
	p.services[svc.ServiceID] = svc
}

// GetService retrieves configuration for a specific Google service.
func (p *GoogleManagedMCPProvider) GetService(serviceID string) (*GoogleServiceMetadata, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	svc, ok := p.services[serviceID]
	if !ok {
		return nil, fmt.Errorf("google mcp service %q not registered", serviceID)
	}
	return svc, nil
}

// ListServices enumerates all registered Google Cloud MCP endpoints.
func (p *GoogleManagedMCPProvider) ListServices() []*GoogleServiceMetadata {
	p.mu.RLock()
	defer p.mu.RUnlock()
	list := make([]*GoogleServiceMetadata, 0, len(p.services))
	for _, svc := range p.services {
		list = append(list, svc)
	}
	return list
}

// NormalizeTool wraps raw Google service tool definitions into standardized MCP tools with risk classifications.
func (p *GoogleManagedMCPProvider) NormalizeTool(serviceID, toolName, description string, risk ToolRiskClass, schema map[string]any) protocol.MCPTool {
	p.mu.Lock()
	defer p.mu.Unlock()

	fullName := fmt.Sprintf("google.%s.%s", serviceID, toolName)
	rawSchema, _ := json.Marshal(schema)
	tool := protocol.MCPTool{
		Name:        fullName,
		Description: fmt.Sprintf("[Google %s] %s", serviceID, description),
		InputSchema: json.RawMessage(rawSchema),
	}
	p.tools[fullName] = tool
	return tool
}

// CheckHealth simulates or probes health of a Google Cloud MCP service.
func (p *GoogleManagedMCPProvider) CheckHealth(ctx context.Context, serviceID string) (string, error) {
	svc, err := p.GetService(serviceID)
	if err != nil {
		return "UNREGISTERED", err
	}
	svc.LastCheckedAt = time.Now().UTC()
	return svc.HealthStatus, nil
}

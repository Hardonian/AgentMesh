package fleet

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ProxyHealthStatus indicates whether a proxy instance is healthy.
type ProxyHealthStatus string

const (
	ProxyHealthy   ProxyHealthStatus = "HEALTHY"
	ProxyDegraded  ProxyHealthStatus = "DEGRADED"
	ProxyUnhealthy ProxyHealthStatus = "UNHEALTHY"
)

// ProxyInstance represents a registered data-plane proxy in GKE, Cloud Run, or VMs.
type ProxyInstance struct {
	InstanceID       string            `json:"instanceId"`
	TenantID         string            `json:"tenantId"`
	Cluster          string            `json:"cluster"`
	Region           string            `json:"region"`
	RuntimeType      string            `json:"runtimeType"` // GKE, CLOUD_RUN, VM
	ProxyVersion     string            `json:"proxyVersion"`
	ActiveConfigHash string            `json:"activeConfigHash"`
	Health           ProxyHealthStatus `json:"health"`
	LastHeartbeat    time.Time         `json:"lastHeartbeat"`
	OutboundOnly     bool              `json:"outboundOnly"`
}

// FleetSummary gives an operational view of proxy distribution and upgrade state.
type FleetSummary struct {
	TenantID        string           `json:"tenantId"`
	TotalProxies    int              `json:"totalProxies"`
	HealthyProxies  int              `json:"healthyProxies"`
	CanaryProxies   int              `json:"canaryProxies"`
	VersionCounts   map[string]int   `json:"versionCounts"`
	Instances       []*ProxyInstance `json:"instances"`
	LastRefreshedAt time.Time        `json:"lastRefreshedAt"`
}

// CachedSignedConfig stores local configuration on the proxy for offline survivability.
type CachedSignedConfig struct {
	ConfigBundleJSON []byte    `json:"configBundleJson"`
	Signature        string    `json:"signature"`
	ConfigVersion    string    `json:"configVersion"`
	DownloadedAt     time.Time `json:"downloadedAt"`
	MaxStaleness     time.Duration `json:"maxStaleness"` // e.g. 24h before fail-closed
}

// IsValidOffline checks whether cached config is safe to execute during a control plane outage.
func (c *CachedSignedConfig) IsValidOffline(now time.Time) (bool, string) {
	if len(c.ConfigBundleJSON) == 0 {
		return false, "cached config bundle is empty"
	}
	if c.MaxStaleness <= 0 {
		c.MaxStaleness = 24 * time.Hour
	}
	age := now.Sub(c.DownloadedAt)
	if age > c.MaxStaleness {
		return false, fmt.Sprintf("cached config is stale (%v > max %v); fail-closed policy enforced", age, c.MaxStaleness)
	}
	return true, "cached signed config is active and valid"
}

// Manager orchestrates private proxy fleets across enterprise clusters.
type Manager struct {
	mu           sync.RWMutex
	instances    map[string]*ProxyInstance // tenant:instanceId -> instance
	canaryConfig map[string]string         // tenant -> target canary version
}

// NewManager creates a proxy fleet manager.
func NewManager() *Manager {
	return &Manager{
		instances:    make(map[string]*ProxyInstance),
		canaryConfig: make(map[string]string),
	}
}

func instanceKey(tenantID, id string) string {
	return tenantID + ":" + id
}

// RegisterHeartbeat processes outbound heartbeats from private proxies.
func (m *Manager) RegisterHeartbeat(inst *ProxyInstance) error {
	if inst == nil || inst.InstanceID == "" || inst.TenantID == "" {
		return errors.New("invalid proxy instance")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	k := instanceKey(inst.TenantID, inst.InstanceID)
	inst.LastHeartbeat = time.Now().UTC()
	if inst.Health == "" {
		inst.Health = ProxyHealthy
	}
	inst.OutboundOnly = true
	m.instances[k] = inst
	return nil
}

// SetCanaryTarget initiates progressive proxy rollout for a tenant.
func (m *Manager) SetCanaryTarget(tenantID, version string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.canaryConfig[tenantID] = version
}

// GetFleetSummary aggregates proxy health and version distribution.
func (m *Manager) GetFleetSummary(tenantID string) *FleetSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now().UTC()
	summary := &FleetSummary{
		TenantID:        tenantID,
		VersionCounts:   make(map[string]int),
		Instances:       make([]*ProxyInstance, 0),
		LastRefreshedAt: now,
	}

	canaryVer := m.canaryConfig[tenantID]

	for _, inst := range m.instances {
		if tenantID == "" || inst.TenantID == tenantID {
			summary.TotalProxies++
			summary.VersionCounts[inst.ProxyVersion]++

			// Check heartbeat freshness (mark UNHEALTHY if > 2m without heartbeat)
			if now.Sub(inst.LastHeartbeat) > 2*time.Minute {
				inst.Health = ProxyUnhealthy
			} else if inst.Health == ProxyHealthy {
				summary.HealthyProxies++
			}

			if canaryVer != "" && inst.ProxyVersion == canaryVer {
				summary.CanaryProxies++
			}

			summary.Instances = append(summary.Instances, inst)
		}
	}

	return summary
}

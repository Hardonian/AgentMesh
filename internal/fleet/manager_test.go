package fleet

import (
	"testing"
	"time"
)

func TestProxyFleetAndOfflineSurvivability(t *testing.T) {
	mgr := NewManager()
	tenant := "acme-corp"

	// 1. Register proxy instances in GKE and Cloud Run
	p1 := &ProxyInstance{
		InstanceID:       "proxy-gke-1",
		TenantID:         tenant,
		Cluster:          "gke-prod-us-central1",
		Region:           "us-central1",
		RuntimeType:      "GKE",
		ProxyVersion:     "1.2.0",
		ActiveConfigHash: "hash-abc",
	}
	p2 := &ProxyInstance{
		InstanceID:       "proxy-run-1",
		TenantID:         tenant,
		Cluster:          "cloudrun-europe-west1",
		Region:           "europe-west1",
		RuntimeType:      "CLOUD_RUN",
		ProxyVersion:     "1.3.0-canary",
		ActiveConfigHash: "hash-def",
	}

	_ = mgr.RegisterHeartbeat(p1)
	_ = mgr.RegisterHeartbeat(p2)
	mgr.SetCanaryTarget(tenant, "1.3.0-canary")

	summary := mgr.GetFleetSummary(tenant)
	if summary.TotalProxies != 2 {
		t.Fatalf("expected 2 proxies, got %d", summary.TotalProxies)
	}
	if summary.HealthyProxies != 2 {
		t.Errorf("expected 2 healthy proxies, got %d", summary.HealthyProxies)
	}
	if summary.CanaryProxies != 1 {
		t.Errorf("expected 1 canary proxy, got %d", summary.CanaryProxies)
	}

	// 2. Test Offline Config Survivability
	now := time.Now().UTC()
	cachedConfig := &CachedSignedConfig{
		ConfigBundleJSON: []byte(`{"routes":[]}`),
		Signature:        "sig-12345",
		ConfigVersion:    "bundle-v1",
		DownloadedAt:     now.Add(-2 * time.Hour),
		MaxStaleness:     24 * time.Hour,
	}

	// 2 hours old with 24h limit -> valid
	valid, msg := cachedConfig.IsValidOffline(now)
	if !valid {
		t.Errorf("expected cached config to survive offline: %s", msg)
	}

	// 25 hours old with 24h limit -> fail closed
	staleConfig := &CachedSignedConfig{
		ConfigBundleJSON: []byte(`{"routes":[]}`),
		DownloadedAt:     now.Add(-25 * time.Hour),
		MaxStaleness:     24 * time.Hour,
	}
	validStale, _ := staleConfig.IsValidOffline(now)
	if validStale {
		t.Errorf("expected stale config to fail-closed after 24h")
	}
}

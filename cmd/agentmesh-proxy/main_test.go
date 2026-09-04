package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentmesh/agentmesh/internal/config"
)

func TestProxyServer_BuildAndEndpoints(t *testing.T) {
	cfg := &config.AppConfig{
		Environment: "development",
		ProxyPort:   9090,
	}

	router, cache, err := buildProxyRouter(cfg)
	if err != nil {
		t.Fatalf("failed to build proxy router: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil proxy cache")
	}

	// 1. Health check
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 on /healthz, got %d", rec.Code)
	}

	// 2. Metrics endpoint
	reqMetrics := httptest.NewRequest("GET", "/metrics", nil)
	recMetrics := httptest.NewRecorder()
	router.ServeHTTP(recMetrics, reqMetrics)

	if recMetrics.Code != http.StatusOK {
		t.Errorf("expected 200 on /metrics, got %d", recMetrics.Code)
	}

	// 3. MCP RPC endpoint (tools/list call)
	rpcBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	reqRPC := httptest.NewRequest("POST", "/mcp/rpc", bytes.NewReader([]byte(rpcBody)))
	reqRPC.Header.Set("Content-Type", "application/json")
	recRPC := httptest.NewRecorder()
	router.ServeHTTP(recRPC, reqRPC)

	if recRPC.Code != http.StatusOK {
		t.Errorf("expected 200 on /mcp/rpc tools/list, got %d", recRPC.Code)
	}
}

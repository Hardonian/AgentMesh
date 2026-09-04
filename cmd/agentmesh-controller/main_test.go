package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentmesh/agentmesh/internal/config"
)

func TestControllerServer_BuildAndHealth(t *testing.T) {
	cfg := &config.AppConfig{
		Environment:  "development",
		HTTPPort:     8080,
		SigningKeyID: "test_key",
	}

	srv, err := buildControllerServer(cfg)
	if err != nil {
		t.Fatalf("failed to build controller server: %v", err)
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 on /healthz, got %d", rec.Code)
	}
}

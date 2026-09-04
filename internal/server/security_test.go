package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/database"
	"github.com/agentmesh/agentmesh/internal/identity"
)

func TestSSRFProtection(t *testing.T) {
	cases := []struct {
		url          string
		allowPrivate bool
		expectErr    bool
		name         string
	}{
		{"http://169.254.169.254/computeMetadata/v1/", false, true, "GCP Metadata IP"},
		{"http://metadata.google.internal/computeMetadata/v1/", false, true, "GCP Metadata Hostname"},
		{"http://127.0.0.1:8080/admin", false, true, "Loopback IPv4"},
		{"http://localhost:8080/", false, true, "Localhost"},
		{"file:///etc/passwd", false, true, "File Scheme"},
		{"gopher://127.0.0.1:70/", false, true, "Gopher Scheme"},
		{"http://10.0.0.5/api", false, true, "Private RFC1918 Denied by Default"},
		{"http://10.0.0.5/api", true, false, "Private RFC1918 Permitted with Flag"},
		{"http://192.168.1.1/router", false, true, "RFC1918 192.168"},
		{"", false, true, "Empty URL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateSafeRemoteURL(tc.url, tc.allowPrivate)
			if tc.expectErr && err == nil {
				t.Fatalf("Expected error for URL %q, got nil", tc.url)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Expected success for URL %q, got error: %v", tc.url, err)
			}
		})
	}
}

func TestAuthMiddlewareTenantIsolation(t *testing.T) {
	store := database.NewMemoryStore()
	ctx := context.Background()

	// Generate key for tenant A
	rawKeyA, credA, err := identity.GenerateAPIKey("tenant-alpha", "agent-1", []string{identity.ScopeAgentsRead}, 1*time.Hour, "Alpha Key")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	_ = store.SaveCredential(ctx, credA)

	// Generate key for tenant B
	rawKeyB, credB, err := identity.GenerateAPIKey("tenant-bravo", "agent-2", []string{identity.ScopeAgentsRead}, 1*time.Hour, "Bravo Key")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	_ = store.SaveCredential(ctx, credB)

	handler := AuthMiddleware(store, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := GetAuthenticatedTenant(r.Context())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(tenant))
	}))

	// Case 1b: Valid Tenant B Key accessing Tenant B
	reqB := httptest.NewRequest("GET", "/api/v1/agents", nil)
	reqB.Header.Set("Authorization", "Bearer "+rawKeyB)
	reqB.Header.Set("X-Tenant-ID", "tenant-bravo")
	recB := httptest.NewRecorder()
	handler.ServeHTTP(recB, reqB)
	if recB.Code != http.StatusOK || recB.Body.String() != "tenant-bravo" {
		t.Fatalf("Expected 200 OK with tenant-bravo, got %d: %s", recB.Code, recB.Body.String())
	}

	// Case 1: Valid Tenant A Key accessing Tenant A
	req1 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req1.Header.Set("Authorization", "Bearer "+rawKeyA)
	req1.Header.Set("X-Tenant-ID", "tenant-alpha")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK || rec1.Body.String() != "tenant-alpha" {
		t.Fatalf("Expected 200 OK with tenant-alpha, got %d: %s", rec1.Code, rec1.Body.String())
	}

	// Case 2: Tenant A Key attempting to spoof Tenant B
	req2 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req2.Header.Set("Authorization", "Bearer "+rawKeyA)
	req2.Header.Set("X-Tenant-ID", "tenant-bravo")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden for cross-tenant spoofing, got %d", rec2.Code)
	}

	// Case 3: Missing Auth header
	req3 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for missing auth, got %d", rec3.Code)
	}

	// Case 4: Invalid/tampered key
	req4 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req4.Header.Set("Authorization", "Bearer mesh_invalidkey12345678901234567890")
	rec4 := httptest.NewRecorder()
	handler.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for invalid key, got %d", rec4.Code)
	}
}

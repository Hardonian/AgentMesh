package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/agentmesh/agentmesh/internal/database"
	"github.com/agentmesh/agentmesh/internal/identity"
)

type contextKey string

const (
	TenantContextKey     contextKey = "agentmesh_tenant"
	CredentialContextKey contextKey = "agentmesh_credential"
	RoleContextKey       contextKey = "agentmesh_role"
)

// Role defines authenticated role within a tenant organization.
type Role string

const (
	RoleOwner     Role = "OWNER"
	RoleAdmin     Role = "ADMIN"
	RoleOperator  Role = "OPERATOR"
	RoleApprover  Role = "APPROVER"
	RoleDeveloper Role = "DEVELOPER"
	RoleViewer    Role = "VIEWER"
)

var (
	ErrUnauthenticated = errors.New("authentication required: missing or invalid bearer token or API key")
	ErrTenantMismatch  = errors.New("forbidden: caller credentials do not have access to requested tenant")
	ErrInsufficientRole = errors.New("forbidden: caller lacks required permission scope")
)

// AuthMiddleware provides cryptographic API key and Bearer token verification,
// enforces strict tenant isolation, and binds the verified tenant to the request context.
func AuthMiddleware(store database.Store, enforceAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exempt open health and readiness endpoints
			path := r.URL.Path
			if path == "/healthz" || path == "/readyz" || path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			// Public badges and read-only passports (when flagged public) can pass through
			if strings.HasSuffix(path, "/badge") {
				ctx := context.WithValue(r.Context(), TenantContextKey, getTenantID(r))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			authHeader := r.Header.Get("Authorization")
			apiKeyHeader := r.Header.Get("X-API-Key")
			requestedTenant := r.Header.Get("X-Tenant-ID")

			rawKey := ""
			if strings.HasPrefix(authHeader, "Bearer ") {
				rawKey = strings.TrimPrefix(authHeader, "Bearer ")
			} else if apiKeyHeader != "" {
				rawKey = apiKeyHeader
			}

			if rawKey == "" {
				if enforceAuth {
					writeJSONError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing authorization header or API key")
					return
				}
				// If not strictly enforced (e.g. dev mode without keys configured), fallback to requested or default tenant
				tenant := requestedTenant
				if tenant == "" {
					tenant = "default"
				}
				ctx := context.WithValue(r.Context(), TenantContextKey, tenant)
				ctx = context.WithValue(ctx, RoleContextKey, RoleOwner)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Validate provided key against database credentials
			hashed := identity.HashKey(rawKey)
			cred, err := store.GetCredentialByHash(r.Context(), hashed)
			if err != nil || cred == nil {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid or unknown API key")
				return
			}

			if err := cred.ValidateKey(rawKey); err != nil {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHENTICATED", err.Error())
				return
			}

			// Tenant Isolation Invariant: If X-Tenant-ID was requested, it MUST match credential tenant
			if requestedTenant != "" && subtle.ConstantTimeCompare([]byte(requestedTenant), []byte(cred.TenantID)) != 1 {
				writeJSONError(w, http.StatusForbidden, "TENANT_ACCESS_DENIED", "cross-tenant operation prohibited")
				return
			}

			// Derive role from credential scopes
			role := RoleViewer
			if cred.HasScope(identity.ScopeAdmin) {
				role = RoleOwner
			} else if cred.HasScope(identity.ScopeAgentsWrite) || cred.HasScope(identity.ScopeApprovalsWrite) {
				role = RoleOperator
			} else if cred.HasScope(identity.ScopeAgentsRead) {
				role = RoleDeveloper
			}

			ctx := context.WithValue(r.Context(), TenantContextKey, cred.TenantID)
			ctx = context.WithValue(ctx, CredentialContextKey, cred)
			ctx = context.WithValue(ctx, RoleContextKey, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAuthenticatedTenant retrieves the verified tenant ID from context.
func GetAuthenticatedTenant(ctx context.Context) string {
	if val, ok := ctx.Value(TenantContextKey).(string); ok && val != "" {
		return val
	}
	return "default"
}

// GetAuthenticatedCredential retrieves the caller's credential from context if present.
func GetAuthenticatedCredential(ctx context.Context) *identity.Credential {
	if val, ok := ctx.Value(CredentialContextKey).(*identity.Credential); ok {
		return val
	}
	return nil
}

// writeJSONError formats standardized JSON API errors.
func writeJSONError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    errorCode,
			"message": message,
		},
	})
}

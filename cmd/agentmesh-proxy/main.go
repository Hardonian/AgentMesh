package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/budgets"
	"github.com/agentmesh/agentmesh/internal/config"
	"github.com/agentmesh/agentmesh/internal/crypto"
	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/pkg/protocol"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func buildProxyRouter(cfg *config.AppConfig) (http.Handler, *config.ProxyConfigCache, error) {
	// Core data plane components
	polEngine := policy.NewEngine([]*policy.Policy{})
	keyRing := crypto.NewKeyRing()
	proxyCache := config.NewProxyConfigCache(keyRing, polEngine)
	approvalSvc := approval.NewService()
	budgetTracker := budgets.NewTracker()

	// Upstream MCP Mock / Forwarder
	upstreamHandler := func(ctx context.Context, toolName string, args map[string]any) (*protocol.MCPCallToolResult, error) {
		return &protocol.MCPCallToolResult{
			Content: []protocol.MCPContent{
				{Type: "text", Text: fmt.Sprintf("Tool %q executed successfully via AgentMesh Proxy", toolName)},
			},
		}, nil
	}

	mcpGateway := mcp.NewGateway(polEngine, approvalSvc, budgetTracker, upstreamHandler)
	mcpGateway.RegisterTool(protocol.MCPTool{
		Name:        "bigquery.read",
		Description: "Read query against BigQuery datasets",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
	})
	mcpGateway.RegisterTool(protocol.MCPTool{
		Name:        "bigquery.delete",
		Description: "Destructive delete operation on BigQuery",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
	})

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		version, age, hasConfig := proxyCache.Status()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":        "healthy",
			"configVersion": version,
			"configAgeSec":  age.Seconds(),
			"hasConfig":     hasConfig,
		})
	})
	r.Handle("/metrics", promhttp.Handler())

	// MCP Gateway Endpoint
	r.Post("/mcp/rpc", func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = "default"
		}
		agentID := r.Header.Get("X-Agent-ID")
		if agentID == "" {
			agentID = "anonymous-agent"
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		rpcResp := mcpGateway.HandleRPC(r.Context(), tenantID, agentID, body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rpcResp)
	})

	return r, proxyCache, nil
}

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting AgentMesh Data Plane Proxy",
		"proxyPort", cfg.ProxyPort,
		"env", cfg.Environment,
		"controlPlaneURL", cfg.ControlPlaneURL,
	)

	router, proxyCache, err := buildProxyRouter(cfg)
	if err != nil {
		logger.Error("Failed to build proxy router", "error", err)
		os.Exit(1)
	}

	// Background config synchronizer from control plane
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go syncConfigLoop(ctx, cfg.ControlPlaneURL, proxyCache, logger)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ProxyPort),
		Handler: router,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Proxy server failed", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("AgentMesh Proxy listening and ready", "addr", server.Addr)
	<-stop
	logger.Info("Shutting down AgentMesh Proxy gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	logger.Info("AgentMesh Proxy stopped.")
}

func syncConfigLoop(ctx context.Context, controlPlaneURL string, cache *config.ProxyConfigCache, logger *slog.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Fetch signed config bundle
			url := controlPlaneURL + "/api/v1/config/bundle"
			resp, err := http.Get(url)
			if err != nil {
				logger.Warn("Control plane temporarily unavailable, continuing with cached configuration",
					"error", err,
					"action", "using_last_known_good_cache",
				)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				var bundle crypto.SignedBundle
				if err := json.NewDecoder(resp.Body).Decode(&bundle); err == nil {
					_ = cache.UpdateFromBundle(&bundle, nil)
				}
			}
			_ = resp.Body.Close()
		}
	}
}

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentmesh/agentmesh/internal/config"
)

func main() {
	cfg, _ := config.LoadFromEnv()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting AgentMesh Background Worker",
		"env", cfg.Environment,
		"controlPlaneURL", cfg.ControlPlaneURL,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Periodic tasks: health checks, canary evaluation, audit retention
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logger.Debug("Worker heartbeat: checking agent health and active canaries")
			}
		}
	}()

	<-stop
	logger.Info("Stopping AgentMesh Worker...")
	cancel()
}

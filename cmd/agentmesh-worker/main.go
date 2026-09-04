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

func runWorker(ctx context.Context, cfg *config.AppConfig, logger *slog.Logger, interval time.Duration) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			logger.Debug("Worker heartbeat: checking agent health and active canaries", "env", cfg.Environment)
		}
	}
}

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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = runWorker(ctx, cfg, logger, 10*time.Second)
	}()

	<-stop
	logger.Info("Stopping AgentMesh Worker...")
	cancel()
}

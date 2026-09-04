package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/config"
)

func TestWorker_RunAndCancel(t *testing.T) {
	cfg := &config.AppConfig{
		Environment: "development",
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	err := runWorker(ctx, cfg, logger, 5*time.Millisecond)
	if err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("expected context cancellation, got %v", err)
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arc/hawk-agent/internal/auth"
	"github.com/arc/hawk-agent/internal/buffer"
	"github.com/arc/hawk-agent/internal/config"
	"github.com/arc/hawk-agent/internal/connectors"
	"github.com/arc/hawk-agent/internal/health"
	"github.com/arc/hawk-agent/internal/scanner"
	"go.uber.org/zap"
)

func main() {
	// Check admin/root privileges before anything else.
	if err := checkPrivileges(); err != nil {
		fmt.Fprintf(os.Stderr, "privilege check failed: %v\n", err)
		os.Exit(1)
	}

	// Load configuration from YAML + env.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize structured logger.
	logger, err := newLogger(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("hawk-agent starting",
		zap.String("agent_id", cfg.AgentID),
		zap.String("server_url", cfg.ServerURL),
		zap.String("data_dir", cfg.DataDir),
	)

	// Initialize the offline SQLite buffer.
	localQueue, err := buffer.NewLocalQueue(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init local queue", zap.Error(err))
	}
	defer localQueue.Close()

	// Initialize Keycloak auth client.
	authClient := auth.NewClient(cfg, logger)

	// Initialize the scanner API connector.
	connector := connectors.NewScannerConnector(cfg, authClient, logger)

	// Initialize the sync loop.
	syncLoop := buffer.NewSyncLoop(cfg, localQueue, connector, logger)

	// Initialize the scan orchestrator.
	scanOrchestrator := scanner.NewOrchestrator(cfg, localQueue, connector, logger)

	// Initialize the health server.
	healthServer := health.NewServer(cfg, localQueue, syncLoop, logger)

	// Root context for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start all subsystems.
	syncLoop.Start(ctx)
	scanOrchestrator.Start(ctx)
	healthServer.Start()

	// SIGHUP reloads config.
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	go func() {
		for range sighup {
			logger.Info("received SIGHUP, reloading config")
			newCfg, err := config.Load()
			if err != nil {
				logger.Error("config reload failed", zap.Error(err))
				continue
			}
			*cfg = *newCfg
			logger.Info("config reloaded successfully")
		}
	}()

	// Wait for SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutdown signal received", zap.String("signal", sig.String()))

	// Graceful shutdown sequence.
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logger.Info("flushing pending syncs before shutdown")
	syncLoop.FlushAndStop(shutdownCtx)

	logger.Info("stopping health server")
	healthServer.Stop(shutdownCtx)

	logger.Info("hawk-agent stopped cleanly")
}

func newLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	return cfg.Build()
}

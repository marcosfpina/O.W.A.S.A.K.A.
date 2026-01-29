package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/network/discovery"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/network/dns"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// App represents the main application
type App struct {
	cfg    *config.Config
	logger *logging.Logger
}

// New creates a new application instance
func New(cfg *config.Config, logger *logging.Logger) *App {
	return &App{
		cfg:    cfg,
		logger: logger,
	}
}

// Run starts the application
func (a *App) Run() error {
	a.logger.Info("Starting O.W.A.S.A.K.A. SIEM...")

	// Create a context that acts as a root for all services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Log configuration summary
	a.logger.Infow("Configuration loaded",
		"environment", os.Getenv("OSWAKA_ENV"),
		"log_level", a.cfg.Logging.Level,
		"server_port", a.cfg.Server.Port,
	)

	// Initialize Services
	dnsService := dns.NewService(&a.cfg.Network.DNS, a.logger)
	discoveryService := discovery.NewService(&a.cfg.Network.Discovery, a.logger)

	// Start Services
	if err := dnsService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start DNS service", "error", err)
		return err
	}

	if err := discoveryService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Discovery service", "error", err)
		// Don't return err here to allow app to run even if discovery fails (critical vs non-critical)
		// Actually, Service.Start already handles graceful failure for permissions, so this is safe.
	}

	a.logger.Info("System ready and waiting for signals (Press Ctrl+C to stop)")

	// Wait for termination signal
	select {
	case sig := <-sigChan:
		a.logger.Infow("Received shutdown signal", "signal", sig)
		// Perform cleanup here

		// Give services a moment to shut down gracefully
		_, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		a.logger.Info("Shutdown complete")
	case <-ctx.Done():
		a.logger.Info("Context cancelled, shutting down")
	}

	return nil
}

package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/analytics/correlation"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/api"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/browser/firefox"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/discovery/attack_surface"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/discovery/physical"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/discovery/virtual"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/network/discovery"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/network/dns"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/storage/db"
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

	// Connect to NATS event bus (optional — nil publisher disables event publishing)
	var pub *events.Publisher
	if a.cfg.NatsURL != "" {
		var err error
		pub, err = events.Connect(a.cfg.NatsURL)
		if err != nil {
			a.logger.Warnw("NATS unavailable, events disabled", "url", a.cfg.NatsURL, "error", err)
		} else {
			a.logger.Infow("NATS connected", "url", a.cfg.NatsURL)
			defer pub.Close()
		}
	}

	// Storage Engine
	database, err := db.New(&a.cfg.Storage.Local, a.logger)
	if err != nil {
		a.logger.Errorw("Failed to initialize database", "error", err)
		return err
	}
	defer database.Close()
	
	repository := db.NewRepository(database)

	// M3 Command Center API (WebSocket/HTTP)
	apiServer := api.NewServer(&a.cfg.Server, a.logger)
	if err := apiServer.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start API Server", "error", err)
	}
	defer apiServer.Stop()

	// Milestone 4: Correlation Engine (Threat Detection)
	engine := correlation.NewEngine(&a.cfg.Analytics.Correlation, a.logger)

	// Form Unified Pipeline
	pipeline := events.NewPipeline(repository, apiServer.Hub, pub, a.logger)

	// Hook Engine into Pipeline
	pipeline.SetEngine(engine)
	engine.SetAlertCallback(pipeline.PushNetworkEvent)

	// Initialize Services
	dnsService := dns.NewService(&a.cfg.Network.DNS, a.logger, pipeline)
	discoveryService := discovery.NewService(&a.cfg.Network.Discovery, a.logger, pipeline)

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

	// M2 Services
	attackSurface := attack_surface.NewService(&a.cfg.Discovery.AttackSurface, a.logger, pipeline)
	if err := attackSurface.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Attack Surface Scanner", "error", err)
	}

	physicalService := physical.NewService(&a.cfg.Discovery.Physical, a.logger, pipeline)
	if err := physicalService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Physical Enumeration service", "error", err)
	}

	virtualService := virtual.NewService(&a.cfg.Discovery.Containers, a.logger, pipeline)
	if err := virtualService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Virtual Discovery service", "error", err)
	}

	// M2 Browser Hardening
	firefoxService := firefox.NewService(&a.cfg.Browser, a.logger)
	if err := firefoxService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Firefox service", "error", err)
	}

	a.logger.Info("System ready and waiting for signals (Press Ctrl+C to stop)")

	// Wait for termination signal
	select {
	case sig := <-sigChan:
		a.logger.Infow("Received shutdown signal", "signal", sig)
		// Perform cleanup here

		// Give services a moment to shut down gracefully
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		<-shutdownCtx.Done()

		a.logger.Info("Shutdown complete")
	case <-ctx.Done():
		a.logger.Info("Context cancelled, shutting down")
	}

	return nil
}

package app

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/analytics/correlation"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/analytics/ml"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/analytics/stream"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/api"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/browser/automation"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/browser/firefox"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/discovery/attack_surface"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/discovery/physical"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/discovery/reconciliation"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/discovery/virtual"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/network/discovery"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/network/dns"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/network/proxy"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/network/topology"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/storage/db"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/storage/integrity"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/storage/nas"
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

	// Stream Processor — normalizes + enriches events with sliding-window context
	streamProc := stream.NewProcessor(&a.cfg.Analytics.Stream, a.logger)
	streamProc.Start(ctx)

	// Milestone 4: Correlation Engine (Threat Detection)
	engine := correlation.NewEngine(&a.cfg.Analytics.Correlation, a.logger)

	// Form Unified Pipeline
	pipeline := events.NewPipeline(repository, apiServer.Hub, pub, a.logger)

	// Hook Engine into Pipeline
	pipeline.SetEngine(engine)
	engine.SetAlertCallback(pipeline.PushNetworkEvent)
	pipeline.SetStreamEnricher(streamProc)

	// ML Anomaly Detector — Isolation Forest + behavioral baselining
	mlService := ml.NewService(&a.cfg.Analytics.ML, a.logger, pipeline)
	if err := mlService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start ML Anomaly Detector", "error", err)
	}
	pipeline.SetEventObserver(mlService)

	// Topology Mapper — builds live network graph from asset/event streams
	topoBuilder := topology.NewBuilder(a.logger)
	topoBuilder.OnChange(func(snap topology.GraphSnapshot) {
		// Push TOPOLOGY_UPDATE to all connected WebSocket clients
		msg := map[string]any{
			"type": "TOPOLOGY_UPDATE",
			"data": topology.ToD3(snap),
		}
		apiServer.Hub.Broadcast(msg)
	})
	pipeline.SetTopologyMapper(topoBuilder)

	// Register REST endpoint for stream processor stats
	apiServer.RegisterHandler("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if err := json.NewEncoder(w).Encode(streamProc.Stats()); err != nil {
			a.logger.Errorw("Failed to encode stream stats", "error", err)
		}
	})

	// Register REST endpoint for full topology snapshot
	apiServer.RegisterHandler("/api/topology", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		snap := topoBuilder.Snapshot()
		if err := json.NewEncoder(w).Encode(topology.ToD3(snap)); err != nil {
			a.logger.Errorw("Failed to encode topology", "error", err)
		}
	})

	if err := apiServer.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start API Server", "error", err)
	}
	defer apiServer.Stop()

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

	virtualService := virtual.NewService(&a.cfg.Discovery.Containers, &a.cfg.Discovery.Virtual, a.logger, pipeline)
	if err := virtualService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Virtual Discovery service", "error", err)
	}

	// Continuous Reconciliation Engine — drift detection
	reconEngine := reconciliation.NewEngine(&a.cfg.Discovery.Reconciliation, repository, pipeline, a.logger)
	if err := reconEngine.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Reconciliation Engine", "error", err)
	}

	// Transparent Proxy Engine — HTTP/HTTPS interception + DPI
	proxyService := proxy.NewService(&a.cfg.Network.Proxy, a.logger, pipeline)
	if err := proxyService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Proxy service", "error", err)
	}
	defer proxyService.Stop()

	// M2 Browser Hardening
	firefoxService := firefox.NewService(&a.cfg.Browser, a.logger)
	if err := firefoxService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Firefox service", "error", err)
	}

	// Browser Automation — CDP forensic logging
	autoService := automation.NewService(&a.cfg.Browser.Automation, a.logger, pipeline, a.cfg.Storage.Local.DataDir)
	if err := autoService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start Browser Automation", "error", err)
	}

	// Integrity Verifier — Merkle trees + immutable audit log
	integrityService, err := integrity.NewService(&a.cfg.Storage.Integrity, repository, pipeline, a.logger)
	if err != nil {
		a.logger.Errorw("Failed to initialize Integrity Verifier", "error", err)
	} else {
		if err := integrityService.Start(ctx); err != nil {
			a.logger.Errorw("Failed to start Integrity Verifier", "error", err)
		}
		defer integrityService.Stop()
	}

	// NAS Connector — air-gapped NFS/SMB storage
	nasService := nas.NewService(&a.cfg.Storage.NAS, a.logger, pipeline)
	if err := nasService.Start(ctx); err != nil {
		a.logger.Errorw("Failed to start NAS Connector", "error", err)
	}
	defer nasService.Stop(ctx)

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

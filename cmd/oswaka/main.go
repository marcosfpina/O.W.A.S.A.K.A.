package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/app"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

var (
	// Build information (injected at build time)
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/examples/default.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("O.W.A.S.A.K.A. SIEM\nVersion: %s\nCommit: %s\nBuilt: %s\n", version, commit, buildTime)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := logging.NewLogger(&cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	// Sync logger on exit
	defer func() {
		_ = logger.Sync()
	}()

	// Log startup information
	logger.Infow("Initializing O.W.A.S.A.K.A.",
		"version", version,
		"commit", commit,
		"build_time", buildTime,
	)

	// Create and start application
	application := app.New(cfg, logger)
	if err := application.Run(); err != nil {
		logger.Fatalw("Application runtime error", "error", err)
	}
}

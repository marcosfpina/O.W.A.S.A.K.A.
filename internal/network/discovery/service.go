package discovery

import (
	"context"
	"fmt"
	"net"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service manages the Discovery scanning
type Service struct {
	cfg     *config.ScanConfig
	logger  *logging.Logger
	scanner *Scanner
}

// NewService creates a new discovery service
func NewService(cfg *config.ScanConfig, logger *logging.Logger, pipeline *events.Pipeline) *Service {
	return &Service{
		cfg:     cfg,
		logger:  logger,
		scanner: NewScanner(cfg, logger, pipeline),
	}
}

// Start starts the discovery service
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Discovery Service is disabled")
		return nil
	}

	// Find suitable interface (for MVP, pick the first non-loopback)
	iface, err := s.findInterface()
	if err != nil {
		s.logger.Warnw("Passive Discovery disabled: no suitable interface found", "error", err)
		return nil // Don't crash, just disable this feature
	}

	s.logger.Infow("Starting Passive Discovery", "interface", iface)

	// Start scanner
	if err := s.scanner.Start(iface); err != nil {
		// If we lack permissions, log a warning but don't crash core app
		s.logger.Warnw("Passive Discovery failed to start (permission issue?)", "error", err)
		return nil
	}

	// Wait for shutdown
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

// Stop stops the discovery service
func (s *Service) Stop() {
	s.scanner.Stop()
}

func (s *Service) findInterface() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, i := range ifaces {
		// Skip loopback and down interfaces
		if i.Flags&net.FlagLoopback != 0 || i.Flags&net.FlagUp == 0 {
			continue
		}
		// Return the first one found (e.g., eth0, wlan0)
		return i.Name, nil
	}

	return "", fmt.Errorf("no active interface found")
}

package virtual

import (
	"context"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service manages the virtual discovery (Containers/Hypervisors)
type Service struct {
	cfg      *config.ContainerConfig
	logger   *logging.Logger
	pipeline *events.Pipeline
	docker   *DockerScanner
}

// NewService creates a new virtual discovery orchestrator
func NewService(cfg *config.ContainerConfig, logger *logging.Logger, pl *events.Pipeline) *Service {
	return &Service{
		cfg:      cfg,
		logger:   logger,
		pipeline: pl,
		docker:   NewDockerScanner(cfg, logger, pl),
	}
}

// Start begins periodic polling of Virtualization substrates
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Virtual Discovery (Containers) is disabled")
		return nil
	}

	s.logger.Info("Starting Virtual Container Discovery Engine")

	go func() {
		// Initial full sweep
		if err := s.docker.Scan(ctx); err != nil {
			s.logger.Warnw("Docker sweep failed (Engine down or permission denied?)", "error", err)
		}

		interval := time.Duration(s.cfg.ScanIntervalMinutes) * time.Minute
		if interval == 0 {
			interval = 30 * time.Minute
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("Stopping Virtual Container Discovery Engine")
				return
			case <-ticker.C:
				if err := s.docker.Scan(ctx); err != nil {
					s.logger.Warnw("Periodic Docker sweep failed", "error", err)
				}
			}
		}
	}()

	return nil
}

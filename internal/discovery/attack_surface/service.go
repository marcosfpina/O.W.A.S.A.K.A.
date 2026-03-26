package attack_surface

import (
	"context"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service manages the attack surface scanner lifecycle
type Service struct {
	cfg      *config.AttackSurfaceConfig
	logger   *logging.Logger
	scanner  *PortScanner
	pipeline *events.Pipeline
}

// NewService creates a new attack surface service
func NewService(cfg *config.AttackSurfaceConfig, logger *logging.Logger, pl *events.Pipeline) *Service {
	return &Service{
		cfg:      cfg,
		logger:   logger,
		scanner:  NewPortScanner(cfg, logger),
		pipeline: pl,
	}
}

// Start begins scanning the local host for open ports
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Attack Surface Scanner is disabled")
		return nil
	}

	target := "127.0.0.1"
	s.logger.Infow("Starting Attack Surface Scanner",
		"target", target,
		"port_start", s.cfg.PortRange.Start,
		"port_end", s.cfg.PortRange.End,
		"workers", s.cfg.ConcurrentScans,
	)

	go func() {
		results := s.scanner.ScanHost(ctx, target)
		for result := range results {
			if s.pipeline != nil {
				s.pipeline.PushNetworkEvent(models.NetworkEvent{
					Type:        models.EventPortScan,
					Source:      "local_scanner",
					Destination: result.Host,
					Metadata: map[string]any{
						"port":   result.Port,
						"open":   result.Open,
						"banner": result.Banner,
					},
				})
			} else {
				s.logger.Infow("Open port detected",
					"host", result.Host,
					"port", result.Port,
					"banner", result.Banner,
				)
			}
		}
		s.logger.Info("Attack Surface Scan complete")
	}()

	return nil
}

// Stop is a no-op since shutdown is handled via context cancellation
func (s *Service) Stop() {
	s.logger.Info("Attack Surface Scanner stopped")
}

package attack_surface

import (
	"context"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// AssetLister provides read access to discovered assets.
type AssetLister interface {
	ListAssets() ([]models.Asset, error)
}

// Service manages the attack surface scanner lifecycle
type Service struct {
	cfg      *config.AttackSurfaceConfig
	logger   *logging.Logger
	scanner  *PortScanner
	pipeline *events.Pipeline
	assets   AssetLister
}

// NewService creates a new attack surface service.
// If assets is non-nil, targets are drawn from discovered assets instead of localhost.
func NewService(cfg *config.AttackSurfaceConfig, logger *logging.Logger, pl *events.Pipeline, assets AssetLister) *Service {
	return &Service{
		cfg:      cfg,
		logger:   logger,
		scanner:  NewPortScanner(cfg, logger),
		pipeline: pl,
		assets:   assets,
	}
}

// Start begins scanning discovered hosts (or localhost as fallback) for open ports
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Attack Surface Scanner is disabled")
		return nil
	}

	targets := s.resolveTargets()

	s.logger.Infow("Starting Attack Surface Scanner",
		"targets", targets,
		"port_start", s.cfg.PortRange.Start,
		"port_end", s.cfg.PortRange.End,
		"workers", s.cfg.ConcurrentScans,
	)

	for _, target := range targets {
		go s.scanTarget(ctx, target)
	}

	return nil
}

func (s *Service) resolveTargets() []string {
	if s.assets != nil {
		assets, err := s.assets.ListAssets()
		if err == nil && len(assets) > 0 {
			seen := make(map[string]bool)
			var targets []string
			for _, a := range assets {
				if a.IP != "" && !seen[a.IP] {
					seen[a.IP] = true
					targets = append(targets, a.IP)
				}
			}
			if len(targets) > 0 {
				return targets
			}
		}
	}
	return []string{"127.0.0.1"}
}

func (s *Service) scanTarget(ctx context.Context, target string) {
	results := s.scanner.ScanHost(ctx, target)
	for result := range results {
		if s.pipeline != nil {
			s.pipeline.PushNetworkEvent(models.NetworkEvent{
				Type:        models.EventPortScan,
				Source:      "attack_surface_scanner",
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
	s.logger.Infow("Attack Surface Scan complete", "target", target)
}

// Stop is a no-op since shutdown is handled via context cancellation
func (s *Service) Stop() {
	s.logger.Info("Attack Surface Scanner stopped")
}

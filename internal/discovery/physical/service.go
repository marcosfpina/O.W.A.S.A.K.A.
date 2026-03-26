package physical

import (
	"context"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service manages the physical hardware enumeration lifecycle
type Service struct {
	cfg        *config.PhysicalConfig
	logger     *logging.Logger
	enumerator *Enumerator
	pipeline   *events.Pipeline
}

// NewService creates a new physical discovery service
func NewService(cfg *config.PhysicalConfig, logger *logging.Logger, pl *events.Pipeline) *Service {
	return &Service{
		cfg:        cfg,
		logger:     logger,
		enumerator: NewEnumerator(cfg, logger, pl),
		pipeline:   pl,
	}
}

// Start runs hardware enumeration once on startup  
// Future: periodic re-scan using s.cfg.ScanIntervalMinutes
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Physical Enumeration is disabled")
		return nil
	}

	s.logger.Info("Starting Physical Hardware Enumeration")
	go s.enumerator.Enumerate(ctx)

	return nil
}

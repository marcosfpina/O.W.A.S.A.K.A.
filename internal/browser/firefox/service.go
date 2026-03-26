package firefox

import (
	"context"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service provides the lifecycle management for the browser module
type Service struct {
	cfg      *config.BrowserConfig
	logger   *logging.Logger
	launcher *Launcher
}

// NewService creates the firefox/browser service
func NewService(cfg *config.BrowserConfig, logger *logging.Logger) *Service {
	return &Service{
		cfg:      cfg,
		logger:   logger,
		launcher: NewLauncher(&cfg.Firefox, logger),
	}
}

// Start launches the browser in a goroutine if it's enabled
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Browser module is disabled")
		return nil
	}

	s.logger.Info("Starting Hardened Browser Service")

	go func() {
		if err := s.launcher.Launch(ctx); err != nil {
			s.logger.Errorw("Browser execution failed", "error", err)
		}
	}()

	return nil
}

// Stop is handled by Context cancellation in app.go
func (s *Service) Stop() {
	s.logger.Info("Browser service stopped")
}

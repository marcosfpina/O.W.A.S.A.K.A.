package proxy

import (
	"context"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service manages the transparent proxy lifecycle.
type Service struct {
	cfg      *config.ProxyConfig
	logger   *logging.Logger
	pipeline *events.Pipeline
	server   *Server
}

// NewService creates a proxy service (does not start it).
func NewService(cfg *config.ProxyConfig, logger *logging.Logger, pipeline *events.Pipeline) *Service {
	return &Service{
		cfg:      cfg,
		logger:   logger,
		pipeline: pipeline,
	}
}

// Start initialises the proxy server and begins accepting connections.
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Proxy service is disabled")
		return nil
	}

	srv, err := NewServer(s.cfg, s.logger, s.pipeline)
	if err != nil {
		return err
	}
	s.server = srv
	return s.server.Start(ctx)
}

// Stop gracefully shuts down the proxy.
func (s *Service) Stop() {
	if s.server != nil {
		s.server.Stop()
	}
}

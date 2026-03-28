package dns

import (
	"context"

	"github.com/miekg/dns"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service manages the DNS server lifecycle
type Service struct {
	cfg      *config.DNSConfig
	logger   *logging.Logger
	resolver *Resolver
	server   *dns.Server
}

// NewService creates a new DNS service
func NewService(cfg *config.DNSConfig, logger *logging.Logger, pipeline *events.Pipeline) *Service {
	resolver := NewResolver(cfg, logger, pipeline)
	return &Service{
		cfg:      cfg,
		logger:   logger,
		resolver: resolver,
		server: &dns.Server{
			Addr:    cfg.ListenAddress,
			Net:     "udp",
			Handler: resolver,
		},
	}
}

// Start starts the DNS server
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("DNS Service is disabled")
		return nil
	}

	s.logger.Infow("Starting DNS Service", "address", s.cfg.ListenAddress)

	// Start DNS cache evictor
	s.resolver.StartCacheEvictor(ctx)

	// Start server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			s.logger.Errorw("DNS Server failed", "error", err)
		}
	}()

	// Wait for context cancellation to stop the server
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

// Stop stops the DNS server
func (s *Service) Stop() {
	if s.server != nil {
		s.logger.Info("Stopping DNS Service...")
		if err := s.server.Shutdown(); err != nil {
			s.logger.Errorw("Failed to shutdown DNS server", "error", err)
		}
	}
}

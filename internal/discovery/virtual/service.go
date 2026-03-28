package virtual

import (
	"context"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service manages the virtual discovery (Containers + Hypervisors)
type Service struct {
	containerCfg *config.ContainerConfig
	vmCfg        *config.VirtualConfig
	logger       *logging.Logger
	pipeline     *events.Pipeline
	docker       *DockerScanner
	libvirt      *LibvirtScanner
	tracker      *ResourceTracker
}

// NewService creates a new virtual discovery orchestrator.
// Pass vmCfg as nil if hypervisor scanning is not configured.
func NewService(containerCfg *config.ContainerConfig, vmCfg *config.VirtualConfig, logger *logging.Logger, pl *events.Pipeline) *Service {
	docker := NewDockerScanner(containerCfg, logger, pl)
	s := &Service{
		containerCfg: containerCfg,
		vmCfg:        vmCfg,
		logger:       logger,
		pipeline:     pl,
		docker:       docker,
		tracker:      NewResourceTracker(docker.client, logger, pl),
	}
	if vmCfg != nil {
		s.libvirt = NewLibvirtScanner(vmCfg, logger, pl)
	}
	return s
}

// Start begins periodic polling of all virtualization substrates.
func (s *Service) Start(ctx context.Context) error {
	if !s.containerCfg.Enabled && (s.vmCfg == nil || !s.vmCfg.Enabled) {
		s.logger.Info("Virtual Discovery is disabled")
		return nil
	}

	s.logger.Info("Starting Virtual Discovery Engine")

	// Docker container discovery + resource stats
	if s.containerCfg.Enabled {
		go s.runDockerLoop(ctx)
	}

	// Libvirt VM discovery
	if s.vmCfg != nil && s.vmCfg.Enabled && s.libvirt.Enabled() {
		go s.runLibvirtLoop(ctx)
	}

	return nil
}

func (s *Service) runDockerLoop(ctx context.Context) {
	// Initial sweep
	if err := s.docker.Scan(ctx); err != nil {
		s.logger.Warnw("Docker sweep failed (Engine down or permission denied?)", "error", err)
	} else {
		// Collect stats right after successful discovery
		if err := s.tracker.CollectAll(ctx); err != nil {
			s.logger.Warnw("Docker stats collection failed", "error", err)
		}
	}

	interval := time.Duration(s.containerCfg.ScanIntervalMinutes) * time.Minute
	if interval == 0 {
		interval = 30 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping Docker Discovery")
			return
		case <-ticker.C:
			if err := s.docker.Scan(ctx); err != nil {
				s.logger.Warnw("Periodic Docker sweep failed", "error", err)
				continue
			}
			if err := s.tracker.CollectAll(ctx); err != nil {
				s.logger.Warnw("Periodic Docker stats failed", "error", err)
			}
		}
	}
}

func (s *Service) runLibvirtLoop(ctx context.Context) {
	// Initial sweep
	if err := s.libvirt.Scan(ctx); err != nil {
		s.logger.Warnw("Libvirt sweep failed (virsh unavailable?)", "error", err)
	}

	interval := time.Duration(s.vmCfg.ScanIntervalMinutes) * time.Minute
	if interval == 0 {
		interval = 60 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping Libvirt Discovery")
			return
		case <-ticker.C:
			if err := s.libvirt.Scan(ctx); err != nil {
				s.logger.Warnw("Periodic Libvirt sweep failed", "error", err)
			}
		}
	}
}

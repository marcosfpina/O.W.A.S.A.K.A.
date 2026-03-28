package nas

import (
	"context"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service manages the NAS storage lifecycle with health monitoring.
type Service struct {
	cfg      *config.NASConfig
	logger   *logging.Logger
	pipeline *events.Pipeline
	mounter  *Mounter
}

// NewService creates a NAS service.
func NewService(cfg *config.NASConfig, logger *logging.Logger, pipeline *events.Pipeline) *Service {
	return &Service{
		cfg:      cfg,
		logger:   logger,
		pipeline: pipeline,
		mounter:  NewMounter(cfg, logger),
	}
}

// Start mounts the NAS share and begins health monitoring.
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("NAS Connector is disabled")
		return nil
	}

	s.logger.Infow("Starting NAS Connector", "type", s.cfg.Type, "host", s.cfg.Host)

	// Attempt mount with retries
	if err := s.mountWithRetry(ctx); err != nil {
		s.logger.Errorw("NAS mount failed after retries", "error", err)
		s.emitHealthEvent("mount_failed", "high")
		return nil // Don't crash the app, NAS is non-critical
	}

	s.emitHealthEvent("mounted", "info")

	// Start health check loop
	go s.healthLoop(ctx)

	return nil
}

// Stop unmounts the NAS share.
func (s *Service) Stop(ctx context.Context) {
	if err := s.mounter.Unmount(ctx); err != nil {
		s.logger.Warnw("NAS unmount error", "error", err)
	}
}

// MountPoint returns the configured mount point path.
func (s *Service) MountPoint() string {
	return s.cfg.MountPoint
}

func (s *Service) mountWithRetry(ctx context.Context) error {
	retries := s.cfg.RetryAttempts
	if retries == 0 {
		retries = 3
	}

	var lastErr error
	for i := 0; i < retries; i++ {
		if err := s.mounter.Mount(ctx); err != nil {
			lastErr = err
			s.logger.Warnw("NAS mount attempt failed", "attempt", i+1, "error", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(i+1) * 5 * time.Second):
			}
			continue
		}
		return nil
	}
	return lastErr
}

func (s *Service) healthLoop(ctx context.Context) {
	interval := time.Duration(s.cfg.HealthCheckIntervalSeconds) * time.Second
	if interval == 0 {
		interval = 60 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	wasHealthy := true

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mounted := s.mounter.IsMounted()
			if mounted && !wasHealthy {
				s.logger.Info("NAS recovered")
				s.emitHealthEvent("recovered", "info")
				wasHealthy = true
			} else if !mounted && wasHealthy {
				s.logger.Warn("NAS mount lost, attempting remount")
				s.emitHealthEvent("mount_lost", "high")
				wasHealthy = false

				// Attempt remount
				if err := s.mounter.Mount(ctx); err != nil {
					s.logger.Errorw("NAS remount failed", "error", err)
				} else {
					wasHealthy = true
					s.emitHealthEvent("remounted", "info")
				}
			}
		}
	}
}

func (s *Service) emitHealthEvent(status, severity string) {
	if s.pipeline == nil {
		return
	}
	s.pipeline.PushNetworkEvent(models.NetworkEvent{
		Type:   models.EventAlert,
		Source: "nas-connector",
		Destination: s.cfg.Host,
		Metadata: map[string]any{
			"nas_type":    s.cfg.Type,
			"mount_point": s.cfg.MountPoint,
			"status":      status,
			"severity":    severity,
		},
		Timestamp: time.Now(),
	})
}

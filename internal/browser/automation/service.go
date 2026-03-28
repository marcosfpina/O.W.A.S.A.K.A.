package automation

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service provides browser automation for forensic logging.
type Service struct {
	cfg      *config.AutomationConfig
	logger   *logging.Logger
	pipeline *events.Pipeline
	cdp      *CDPClient
	dataDir  string // where screenshots/HARs are stored
}

// NewService creates the automation service.
func NewService(cfg *config.AutomationConfig, logger *logging.Logger, pipeline *events.Pipeline, dataDir string) *Service {
	port := cfg.WebDriverPort
	if port == 0 {
		port = 9222
	}
	return &Service{
		cfg:      cfg,
		logger:   logger,
		pipeline: pipeline,
		cdp:      NewCDPClient(port, logger),
		dataDir:  dataDir,
	}
}

// Start attempts to connect to the browser's CDP endpoint and begins monitoring.
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Browser Automation is disabled")
		return nil
	}

	s.logger.Info("Starting Browser Automation Service")

	// Ensure data directory exists
	screenshotDir := filepath.Join(s.dataDir, "screenshots")
	if err := os.MkdirAll(screenshotDir, 0700); err != nil {
		return fmt.Errorf("failed to create screenshot dir: %w", err)
	}

	go s.connectLoop(ctx)
	return nil
}

// connectLoop retries CDP connection until the browser is available.
func (s *Service) connectLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := s.cdp.Connect(ctx); err != nil {
			s.logger.Debugw("CDP not available yet, retrying...", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
				continue
			}
		}

		// Connected — enable monitoring
		if s.cfg.HARLogging {
			if err := s.cdp.EnableNetwork(ctx); err != nil {
				s.logger.Warnw("Failed to enable network monitoring", "error", err)
			}
		}

		s.logger.Info("Browser Automation connected and monitoring")

		// Wait for context cancellation
		<-ctx.Done()
		s.cdp.Close()
		return
	}
}

// TakeScreenshot captures a screenshot and saves it to disk.
// Called externally when a threat alert is triggered.
func (s *Service) TakeScreenshot(ctx context.Context, reason string) (string, error) {
	if s.cdp.conn == nil {
		return "", fmt.Errorf("CDP not connected")
	}

	data, err := s.cdp.CaptureScreenshot(ctx)
	if err != nil {
		return "", err
	}

	// Decode base64 PNG
	pngData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return "", fmt.Errorf("screenshot decode failed: %w", err)
	}

	filename := fmt.Sprintf("screenshot_%s_%s.png",
		time.Now().Format("20060102_150405"),
		sanitizeFilename(reason),
	)
	path := filepath.Join(s.dataDir, "screenshots", filename)

	if err := os.WriteFile(path, pngData, 0600); err != nil {
		return "", err
	}

	// Emit forensic event
	if s.pipeline != nil {
		s.pipeline.PushNetworkEvent(models.NetworkEvent{
			Type:   models.EventAlert,
			Source: "browser-automation",
			Destination: "screenshot",
			Metadata: map[string]any{
				"reason":   reason,
				"path":     path,
				"size":     len(pngData),
				"severity": "info",
			},
			Timestamp: time.Now(),
		})
	}

	s.logger.Infow("Screenshot captured", "path", path, "reason", reason)
	return path, nil
}

// GetHistory retrieves navigation history for forensic analysis.
func (s *Service) GetHistory(ctx context.Context) ([]byte, error) {
	if s.cdp.conn == nil {
		return nil, fmt.Errorf("CDP not connected")
	}
	result, err := s.cdp.GetNavigationHistory(ctx)
	if err != nil {
		return nil, err
	}
	return []byte(result), nil
}

func sanitizeFilename(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, c)
		}
	}
	if len(result) > 50 {
		result = result[:50]
	}
	return string(result)
}

package firefox

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Launcher manages the Firefox process lifecycle
type Launcher struct {
	cfg     *config.FirefoxConfig
	logger  *logging.Logger
	profile *ProfileManager
	policy  *PolicyEnforcer
}

// NewLauncher creates a new browser launcher
func NewLauncher(cfg *config.FirefoxConfig, browserCfg *config.BrowserConfig, logger *logging.Logger) *Launcher {
	return &Launcher{
		cfg:     cfg,
		logger:  logger,
		profile: NewProfileManager(cfg),
		policy:  NewPolicyEnforcer(browserCfg, logger),
	}
}

// Launch starts Firefox with the given context and blocks until exit
func (l *Launcher) Launch(ctx context.Context) error {
	binPath := l.cfg.BinaryPath
	if binPath == "" {
		binPath = "firefox" // Fallback to PATH lookup
	}

	// 1. Generate temp profile
	profileDir, err := l.profile.CreateTempProfile()
	if err != nil {
		return fmt.Errorf("profile generation failed: %w", err)
	}
	defer func() {
		if err := l.profile.Cleanup(profileDir); err != nil {
			l.logger.Warnw("Failed to cleanup firefox profile", "dir", profileDir, "error", err)
		}
	}()

	// 2. Apply enterprise policies (cannot be overridden by user)
	if err := l.policy.Apply(profileDir); err != nil {
		l.logger.Warnw("Failed to apply enterprise policies", "error", err)
	}

	l.logger.Infow("Launching Hardened Firefox", "binary", binPath, "profile", profileDir)

	// 3. Build command arguments
	args := []string{
		"--profile", profileDir,
		"--no-remote",
		"--new-instance",
	}

	// 4. Execute process bound strictly to the provided context
	cmd := exec.CommandContext(ctx, binPath, args...)

	// We pipe stdout/err to null to keep SIEM logs clean from Firefox's noisy GTK logs
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		// Context cancellation returns a known error, we can ignore it if we expect shutdown
		if ctx.Err() != nil {
			l.logger.Info("Firefox process terminated by context cancellation")
			return nil
		}
		return fmt.Errorf("firefox exited with error: %w", err)
	}

	l.logger.Info("Firefox process exited normally")
	return nil
}

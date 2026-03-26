package firefox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
)

// ProfileManager handles the creation of a temporary, hardened Firefox profile
type ProfileManager struct {
	cfg *config.FirefoxConfig
}

// NewProfileManager creates a new ProfileManager
func NewProfileManager(cfg *config.FirefoxConfig) *ProfileManager {
	return &ProfileManager{cfg: cfg}
}

// CreateTempProfile builds a temporary directory with a hardened user.js
func (p *ProfileManager) CreateTempProfile() (string, error) {
	// Either use configured dir or create a new temp one
	profileDir := p.cfg.ProfileDir
	if profileDir == "" {
		dir, err := os.MkdirTemp("", "owasaka-firefox-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp profile dir: %w", err)
		}
		profileDir = dir
	} else {
		if err := os.MkdirAll(profileDir, 0700); err != nil {
			return "", fmt.Errorf("failed to create configured profile dir: %w", err)
		}
	}

	// Write the hardened user.js
	userJSPath := filepath.Join(profileDir, "user.js")
	content := p.generateUserJS()
	
	if err := os.WriteFile(userJSPath, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("failed to write user.js: %w", err)
	}

	return profileDir, nil
}

// Cleanup removes a temporary profile directory
func (p *ProfileManager) Cleanup(profileDir string) error {
	// Only cleanup if it wasn't statically configured, or if we want to wipe it
	if p.cfg.ProfileDir == "" {
		return os.RemoveAll(profileDir)
	}
	return nil
}

// generateUserJS builds the arkenfox-like configuration script
func (p *ProfileManager) generateUserJS() string {
	var prefs []string

	// Basic telemetry and data collection opt-outs
	if p.cfg.TelemetryDisabled {
		prefs = append(prefs, 
			`user_pref("toolkit.telemetry.enabled", false);`,
			`user_pref("toolkit.telemetry.server", "data:,");`,
			`user_pref("toolkit.telemetry.unified", false);`,
			`user_pref("datareporting.healthreport.uploadEnabled", false);`,
			`user_pref("datareporting.policy.dataSubmissionEnabled", false);`,
		)
	}

	// Hardening preferences (WebRTC, isolation)
	if p.cfg.HardeningEnabled {
		prefs = append(prefs,
			`user_pref("media.peerconnection.enabled", false);`, // Disable WebRTC to prevent IP leak
			`user_pref("privacy.firstparty.isolate", true);`,    // First-party isolation
			`user_pref("browser.safebrowsing.enabled", false);`, // Stop Google ping
			`user_pref("browser.safebrowsing.downloads.enabled", false);`,
			`user_pref("browser.newtabpage.activity-stream.feeds.telemetry", false);`,
			`user_pref("network.http.referer.XOriginPolicy", 2);`,
		)
	}

	// Strict proxy routing inside OWASAKA
	if p.cfg.ExtensionLockdown {
		// Example: Route DNS to OWASAKA internal resolver natively via DoH format if supported
		// Or force proxy usage
		prefs = append(prefs,
			`user_pref("network.proxy.type", 0);`, // Direct connection for now unless proxy is configured
		)
	}

	return strings.Join(prefs, "\n")
}

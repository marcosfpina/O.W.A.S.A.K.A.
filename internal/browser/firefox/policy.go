package firefox

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// PolicyEnforcer generates and applies Mozilla Enterprise Policies (policies.json).
// Unlike user.js, enterprise policies cannot be overridden by the user.
type PolicyEnforcer struct {
	cfg    *config.BrowserConfig
	logger *logging.Logger
}

// NewPolicyEnforcer creates a policy enforcer.
func NewPolicyEnforcer(cfg *config.BrowserConfig, logger *logging.Logger) *PolicyEnforcer {
	return &PolicyEnforcer{cfg: cfg, logger: logger}
}

// Apply writes policies.json into the profile's distribution directory.
func (pe *PolicyEnforcer) Apply(profileDir string) error {
	// Mozilla reads policies from {profile}/distribution/policies.json
	distDir := filepath.Join(profileDir, "distribution")
	if err := os.MkdirAll(distDir, 0700); err != nil {
		return err
	}

	policies := pe.buildPolicies()
	data, err := json.MarshalIndent(policies, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(distDir, "policies.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	pe.logger.Infow("Enterprise policies applied", "path", path)
	return nil
}

// policiesRoot is the top-level structure Mozilla expects.
type policiesRoot struct {
	Policies map[string]any `json:"policies"`
}

func (pe *PolicyEnforcer) buildPolicies() policiesRoot {
	policies := make(map[string]any)

	// Telemetry and data collection lockdown
	if pe.cfg.Firefox.TelemetryDisabled {
		policies["DisableTelemetry"] = true
		policies["DisableFirefoxStudies"] = true
		policies["DisableDefaultBrowserAgent"] = true
		policies["OverrideFirstRunPage"] = ""
		policies["OverridePostUpdatePage"] = ""
	}

	// Security hardening
	if pe.cfg.Firefox.HardeningEnabled {
		policies["DisableFormHistory"] = true
		policies["DisablePasswordReveal"] = true
		policies["DisablePocket"] = true
		policies["DisableSecurityBypass"] = map[string]bool{
			"InvalidCertificate":  false,
			"SafeBrowsing":       false,
		}
		policies["EnableTrackingProtection"] = map[string]any{
			"Value":          true,
			"Locked":         true,
			"Cryptomining":   true,
			"Fingerprinting": true,
		}
		policies["HttpsOnlyMode"] = "force_enabled"
		policies["Cookies"] = map[string]any{
			"Behavior":         "reject-tracker-and-partition-foreign",
			"BehaviorPrivate":  "reject",
			"Locked":           true,
		}
		policies["SanitizeOnShutdown"] = map[string]bool{
			"Cache":          true,
			"Cookies":        true,
			"Downloads":      true,
			"FormData":       true,
			"History":        true,
			"Sessions":       true,
			"SiteSettings":   true,
			"OfflineApps":    true,
		}
		policies["DNSOverHTTPS"] = map[string]any{
			"Enabled": false, // We use OWASAKA's own DNS resolver
			"Locked":  true,
		}
	}

	// Extension lockdown
	if pe.cfg.Firefox.ExtensionLockdown {
		policies["ExtensionSettings"] = map[string]any{
			"*": map[string]any{
				"installation_mode": "blocked",
			},
		}
		policies["DisableBuiltinPDFViewer"] = true
	}

	// Proxy routing through OWASAKA transparent proxy
	policies["Proxy"] = pe.buildProxyPolicy()

	// Disable features that leak data
	policies["DisableFirefoxAccounts"] = true
	policies["DisableProfileImport"] = true
	policies["DontCheckDefaultBrowser"] = true
	policies["NoDefaultBookmarks"] = true
	policies["OfferToSaveLogins"] = false
	policies["PasswordManagerEnabled"] = false

	// Preferences that supplement the policies
	policies["Preferences"] = map[string]any{
		"media.peerconnection.enabled":                   map[string]any{"Value": false, "Status": "locked"},
		"geo.enabled":                                    map[string]any{"Value": false, "Status": "locked"},
		"dom.battery.enabled":                            map[string]any{"Value": false, "Status": "locked"},
		"beacon.enabled":                                 map[string]any{"Value": false, "Status": "locked"},
		"browser.send_pings":                             map[string]any{"Value": false, "Status": "locked"},
		"network.http.speculative-parallel-limit":        map[string]any{"Value": 0, "Status": "locked"},
		"privacy.resistFingerprinting":                   map[string]any{"Value": true, "Status": "locked"},
		"privacy.trackingprotection.fingerprinting.enabled": map[string]any{"Value": true, "Status": "locked"},
		"privacy.trackingprotection.cryptomining.enabled":   map[string]any{"Value": true, "Status": "locked"},
	}

	return policiesRoot{Policies: policies}
}

func (pe *PolicyEnforcer) buildProxyPolicy() map[string]any {
	// Default: no proxy override (direct connection)
	proxy := map[string]any{
		"Mode":   "none",
		"Locked": true,
	}

	// If OWASAKA proxy is likely enabled, route through it
	// The user configures the actual address in network.proxy.listen_address
	// Here we just set the policy structure; the profile manager fills in the address
	return proxy
}

// ApplyProxyRouting updates the proxy policy to route through a specific address.
func (pe *PolicyEnforcer) ApplyProxyRouting(profileDir, proxyAddr string) error {
	distDir := filepath.Join(profileDir, "distribution")
	policyPath := filepath.Join(distDir, "policies.json")

	data, err := os.ReadFile(policyPath)
	if err != nil {
		return err
	}

	var root policiesRoot
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}

	root.Policies["Proxy"] = map[string]any{
		"Mode":     "manual",
		"HTTPProxy": proxyAddr,
		"SSLProxy":  proxyAddr,
		"Locked":    true,
	}

	updated, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(policyPath, updated, 0600)
}

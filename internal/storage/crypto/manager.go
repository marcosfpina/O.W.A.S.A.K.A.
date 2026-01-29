package crypto

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Manager handles the lifecycle of the Master Key
type Manager struct {
	cfg     *config.EncryptionConfig
	logger  *logging.Logger
	keyPath string
}

// NewManager creates a new key manager
func NewManager(cfg *config.EncryptionConfig, logger *logging.Logger) *Manager {
	return &Manager{
		cfg:     cfg,
		logger:  logger,
		keyPath: cfg.KeyFile,
	}
}

// GetMasterKey loads or creates the master key
// For MVP, if no key exists, we create a random one and save it.
// In production, this would interact with a KMS or require a passphrase.
func (m *Manager) GetMasterKey() ([]byte, error) {
	if !m.cfg.Enabled {
		return nil, fmt.Errorf("encryption disabled")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.keyPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Try to load existing key
	if key, err := os.ReadFile(m.keyPath); err == nil {
		if len(key) != 32 {
			return nil, fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(key))
		}
		m.logger.Info("Loaded existing Master Key")
		return key, nil
	}

	// Create new key
	m.logger.Warn("No Master Key found. Generating new secure key...")
	key, err := GenerateRandomBytes(32)
	if err != nil {
		return nil, err
	}

	// Save with strict permissions (0600)
	if err := os.WriteFile(m.keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("failed to save master key: %w", err)
	}

	m.logger.Infow("New Master Key generated and saved", "path", m.keyPath)
	return key, nil
}

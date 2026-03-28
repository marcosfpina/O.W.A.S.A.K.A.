package integrity

import (
	"context"
	"encoding/json"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/storage/db"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service provides integrity verification for stored data.
type Service struct {
	cfg       *config.IntegrityConfig
	repo      *db.Repository
	pipeline  *events.Pipeline
	logger    *logging.Logger
	auditLog  *AuditLog
	lastRoot  string // last Merkle root for comparison
}

// NewService creates an integrity verification service.
func NewService(cfg *config.IntegrityConfig, repo *db.Repository, pipeline *events.Pipeline, logger *logging.Logger) (*Service, error) {
	var al *AuditLog
	if cfg.AuditLog != "" {
		var err error
		al, err = NewAuditLog(cfg.AuditLog)
		if err != nil {
			return nil, err
		}
	}

	return &Service{
		cfg:      cfg,
		repo:     repo,
		pipeline: pipeline,
		logger:   logger,
		auditLog: al,
	}, nil
}

// Start begins periodic integrity verification.
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Integrity Verifier is disabled")
		return nil
	}

	s.logger.Info("Starting Integrity Verifier")

	// Initial snapshot
	s.takeSnapshot()

	interval := time.Duration(s.cfg.SnapshotIntervalHours) * time.Hour
	if interval == 0 {
		interval = 1 * time.Hour
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.logger.Info("Stopping Integrity Verifier")
				return
			case <-ticker.C:
				s.takeSnapshot()
				s.verifyAuditLog()
			}
		}
	}()

	return nil
}

// Stop closes the audit log.
func (s *Service) Stop() {
	if s.auditLog != nil {
		s.auditLog.Close()
	}
}

// LogAction writes an entry to the immutable audit log.
func (s *Service) LogAction(action, subject, details string) {
	if s.auditLog == nil {
		return
	}
	if err := s.auditLog.Append(action, subject, details); err != nil {
		s.logger.Errorw("Audit log write failed", "error", err)
	}
}

func (s *Service) takeSnapshot() {
	if !s.cfg.MerkleTree {
		return
	}

	assets, err := s.repo.ListAssets()
	if err != nil {
		s.logger.Errorw("Integrity snapshot: failed to list assets", "error", err)
		return
	}

	// Serialize each asset as a data block
	blocks := make([][]byte, len(assets))
	for i, a := range assets {
		data, _ := json.Marshal(a)
		blocks[i] = data
	}

	tree := BuildTree(blocks)
	root := tree.RootHex()

	if s.lastRoot != "" && s.lastRoot != root {
		s.logger.Warnw("Integrity: Merkle root changed (expected drift from reconciliation)",
			"prev_root", s.lastRoot,
			"new_root", root,
			"assets", len(assets),
		)
	} else {
		s.logger.Infow("Integrity snapshot taken",
			"root", root,
			"assets", len(assets),
		)
	}

	if s.auditLog != nil {
		s.auditLog.Append("snapshot", "merkle_tree", root)
	}

	s.lastRoot = root
}

func (s *Service) verifyAuditLog() {
	if s.cfg.AuditLog == "" {
		return
	}

	if err := Verify(s.cfg.AuditLog); err != nil {
		s.logger.Errorw("AUDIT LOG INTEGRITY VIOLATION", "error", err)
		if s.pipeline != nil {
			s.pipeline.PushNetworkEvent(models.NetworkEvent{
				Type:   models.EventAlert,
				Source: "integrity-verifier",
				Destination: "audit-log",
				Metadata: map[string]any{
					"error":    err.Error(),
					"severity": "critical",
					"log_path": s.cfg.AuditLog,
				},
				Timestamp: time.Now(),
			})
		}
	} else {
		s.logger.Debug("Audit log integrity verified")
	}
}

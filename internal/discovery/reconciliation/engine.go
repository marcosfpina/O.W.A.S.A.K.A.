package reconciliation

import (
	"context"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/storage/db"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Engine performs continuous reconciliation between observed and stored asset state.
type Engine struct {
	cfg      *config.ReconciliationConfig
	repo     *db.Repository
	pipeline *events.Pipeline
	logger   *logging.Logger
	prev     []models.Asset // last snapshot for drift comparison
}

// NewEngine creates a reconciliation engine.
func NewEngine(cfg *config.ReconciliationConfig, repo *db.Repository, pipeline *events.Pipeline, logger *logging.Logger) *Engine {
	return &Engine{
		cfg:      cfg,
		repo:     repo,
		pipeline: pipeline,
		logger:   logger,
	}
}

// Start begins periodic reconciliation checks.
func (e *Engine) Start(ctx context.Context) error {
	if !e.cfg.Enabled {
		e.logger.Info("Reconciliation Engine is disabled")
		return nil
	}

	e.logger.Info("Starting Continuous Reconciliation Engine")

	go func() {
		// Initial snapshot — no diff on first run, just baseline
		if assets, err := e.repo.ListAssets(); err == nil {
			e.prev = assets
			e.logger.Infow("Reconciliation baseline captured", "assets", len(assets))
		}

		interval := time.Duration(e.cfg.CheckIntervalMinutes) * time.Minute
		if interval == 0 {
			interval = 15 * time.Minute
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				e.logger.Info("Stopping Reconciliation Engine")
				return
			case <-ticker.C:
				e.reconcile(ctx)
			}
		}
	}()

	return nil
}

func (e *Engine) reconcile(_ context.Context) {
	curr, err := e.repo.ListAssets()
	if err != nil {
		e.logger.Errorw("Reconciliation failed to list assets", "error", err)
		return
	}

	changes := Diff(e.prev, curr)
	if len(changes) == 0 {
		e.logger.Debugw("Reconciliation: no drift detected", "assets", len(curr))
		e.prev = curr
		return
	}

	e.logger.Infow("Reconciliation drift detected",
		"changes", len(changes),
		"added", countByType(changes, ChangeAdded),
		"removed", countByType(changes, ChangeRemoved),
		"modified", countByType(changes, ChangeModified),
	)

	if e.cfg.AlertOnChange {
		for _, ch := range changes {
			e.pipeline.PushNetworkEvent(models.NetworkEvent{
				Type:   models.EventAlert,
				Source: "reconciliation",
				Destination: ch.AssetID,
				Metadata: map[string]any{
					"change_type": string(ch.Type),
					"asset_id":    ch.AssetID,
					"fields":      ch.Fields,
					"severity":    severityFor(ch.Type),
				},
				Timestamp: time.Now(),
			})
		}
	}

	// Track history by pushing a summary event
	if e.cfg.TrackHistory {
		e.pipeline.PushNetworkEvent(models.NetworkEvent{
			Type:   models.EventVM,
			Source: "reconciliation",
			Destination: "drift-summary",
			Metadata: map[string]any{
				"total_changes": len(changes),
				"added":         countByType(changes, ChangeAdded),
				"removed":       countByType(changes, ChangeRemoved),
				"modified":      countByType(changes, ChangeModified),
				"total_assets":  len(curr),
			},
			Timestamp: time.Now(),
		})
	}

	e.prev = curr
}

func countByType(changes []Change, ct ChangeType) int {
	n := 0
	for _, c := range changes {
		if c.Type == ct {
			n++
		}
	}
	return n
}

func severityFor(ct ChangeType) string {
	switch ct {
	case ChangeRemoved:
		return "high"
	case ChangeModified:
		return "medium"
	default:
		return "low"
	}
}

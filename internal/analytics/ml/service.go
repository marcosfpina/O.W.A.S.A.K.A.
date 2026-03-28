package ml

import (
	"context"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Service provides ML-based anomaly detection.
// It observes events via the pipeline, builds behavioral baselines,
// trains an Isolation Forest, and flags anomalous activity.
type Service struct {
	cfg      *config.MLConfig
	logger   *logging.Logger
	pipeline *events.Pipeline
	forest   *IsolationForest
	baseline *Baseline
}

// NewService creates an ML anomaly detection service.
func NewService(cfg *config.MLConfig, logger *logging.Logger, pipeline *events.Pipeline) *Service {
	return &Service{
		cfg:      cfg,
		logger:   logger,
		pipeline: pipeline,
		forest:   NewIsolationForest(100, 10, cfg.AnomalyThreshold),
		baseline: NewBaseline(cfg.BaselineLearningDays),
	}
}

// Start begins the training and detection loops.
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("ML Anomaly Detector is disabled")
		return nil
	}

	s.logger.Infow("Starting ML Anomaly Detector",
		"threshold", s.cfg.AnomalyThreshold,
		"baseline_days", s.cfg.BaselineLearningDays,
	)

	// Periodic re-training
	trainingInterval := time.Duration(s.cfg.TrainingIntervalHours) * time.Hour
	if trainingInterval == 0 {
		trainingInterval = 6 * time.Hour
	}

	go func() {
		ticker := time.NewTicker(trainingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.train()
			}
		}
	}()

	return nil
}

// Observe records an event for baseline learning and anomaly detection.
// Called by the pipeline for each network event.
func (s *Service) Observe(e models.NetworkEvent) {
	// Always update baseline
	s.baseline.Record(e)

	// Don't score during learning phase
	if s.baseline.IsLearning() {
		return
	}

	// Don't score if forest isn't trained yet
	if !s.forest.Trained() {
		return
	}

	// Don't score alert events (avoid feedback loop)
	if e.Type == models.EventAlert {
		return
	}

	features := s.baseline.FeatureVector(e.Source, e.Timestamp.Hour())
	if s.forest.IsAnomaly(features) {
		score := s.forest.Score(features)
		s.logger.Warnw("ML anomaly detected",
			"source", e.Source,
			"event_type", e.Type,
			"score", score,
			"features", features,
		)
		s.emitAlert(e, score, features)
	}
}

func (s *Service) train() {
	data := s.baseline.TrainingData()
	if len(data) < 10 {
		s.logger.Debugw("ML: insufficient data for training", "samples", len(data))
		return
	}

	s.forest.Train(data)
	s.logger.Infow("ML model retrained", "samples", len(data))
}

func (s *Service) emitAlert(e models.NetworkEvent, score float64, features []float64) {
	if s.pipeline == nil {
		return
	}
	s.pipeline.PushNetworkEvent(models.NetworkEvent{
		Type:   models.EventAlert,
		Source: "ml-anomaly-detector",
		Destination: e.Source,
		Metadata: map[string]any{
			"anomaly_score":    score,
			"original_type":    string(e.Type),
			"features":         features,
			"severity":         severityFromScore(score),
			"detection_method": "isolation_forest",
		},
		Timestamp: time.Now(),
	})
}

func severityFromScore(score float64) string {
	switch {
	case score > 0.8:
		return "critical"
	case score > 0.7:
		return "high"
	case score > 0.6:
		return "medium"
	default:
		return "low"
	}
}

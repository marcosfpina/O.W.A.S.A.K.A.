package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/api"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/storage/db"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// CorrelationEngine acts as a non-blocking hook to inspect passing objects for anomalies
type CorrelationEngine interface {
	Analyze(event models.NetworkEvent)
	AnalyzeAsset(asset models.Asset)
}

// Pipeline operates as a universal bus unifying physical persistence, Web UI pushing, and NATS brokering
type Pipeline struct {
	repo   *db.Repository
	hub    *api.WSHub
	pub    *Publisher
	logger *logging.Logger
	engine CorrelationEngine
}

// NewPipeline constructs an event dispatcher bridging all output formats
func NewPipeline(repo *db.Repository, hub *api.WSHub, pub *Publisher, logger *logging.Logger) *Pipeline {
	return &Pipeline{
		repo:   repo,
		hub:    hub,
		pub:    pub,
		logger: logger,
	}
}

// SetEngine dynamically binds a Correlation module onto the live pipeline layer
func (p *Pipeline) SetEngine(engine CorrelationEngine) {
	p.engine = engine
}

// PushNetworkEvent accepts an event structure and dispatches globally
func (p *Pipeline) PushNetworkEvent(e models.NetworkEvent) {
	// 1. Hydrate defaults safely
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	// 2. Persist cleanly to disk locally via BoltDB
	if p.repo != nil {
		if err := p.repo.LogEvent(&e); err != nil {
			p.logger.Errorw("Failed to flush event to storage", "error", err, "event_id", e.ID)
		}
	}

	// 3. Forward JSON straight to Svelte Web UI socket
	if p.hub != nil {
		p.hub.Broadcast(e)
	}

	// 4. (Optional) Stream into NATS for inter-application architectures
	if p.pub != nil {
		out := Event{
			EventID:       e.ID,
			EventType:     string(e.Type),
			Timestamp:     e.Timestamp,
			SourceService: "SIEM",
			Payload:       e.Metadata,
		}
		
		// embed intrinsic data 
		if out.Payload == nil {
			out.Payload = make(map[string]any)
		}
		out.Payload["source"] = e.Source
		out.Payload["destination"] = e.Destination

		p.pub.Publish("events.network."+string(e.Type), out)
	}

	// 5. Fire un-blocking analysis asynchronously against the Threat module
	if p.engine != nil && e.Type != models.EventAlert {
		go p.engine.Analyze(e)
	}
}

// PushAsset records hardware configurations and network nodes to BoltDB and UI
func (p *Pipeline) PushAsset(a models.Asset) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	
	if a.FirstSeen.IsZero() {
		a.FirstSeen = time.Now()
	}
	a.LastSeen = time.Now()

	// 1. Persist definitively to BoltDB key store
	if p.repo != nil {
		if err := p.repo.SaveAsset(&a); err != nil {
			p.logger.Errorw("Failed to save asset entity", "error", err, "asset_id", a.ID)
		}
	}

	// 2. Stream to GUI topology graph via WebSocket Hub
	if p.hub != nil {
		// Wrap as an 'asset' discovery envelope so Svelte knows what it is
		envelope := map[string]any{
			"type": "ASSET_DISCOVERY",
			"data": a,
			"timestamp": time.Now(),
		}
		
		p.hub.Broadcast(envelope)
	}

	// 3. Inform external services consuming NATS optionally
	if p.pub != nil {
		data, _ := json.Marshal(a)
		payload := map[string]any{"asset": string(data)}
		p.pub.Publish("events.topology.asset", Event{
			EventID:       a.ID,
			EventType:     "ASSET",
			Timestamp:     time.Now(),
			Payload:       payload,
		})
	}
}

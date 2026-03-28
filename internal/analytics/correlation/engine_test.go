package correlation

import (
	"testing"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

func testLogger() *logging.Logger {
	cfg := &config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}
	l, _ := logging.NewLogger(cfg)
	return l
}

func newTestEngine() *Engine {
	cfg := &config.CorrelationConfig{Enabled: true}
	return &Engine{
		cfg:    cfg,
		logger: testLogger(),
		rules:  DefaultRules(),
	}
}

func TestEngine_AnalyzeDNSExfiltration(t *testing.T) {
	var alerts []models.NetworkEvent
	e := newTestEngine()
	e.onAlert = func(ev models.NetworkEvent) { alerts = append(alerts, ev) }

	// Benign query — should NOT trigger
	e.Analyze(models.NetworkEvent{
		ID:   "1",
		Type: models.EventDNS,
		Metadata: map[string]any{
			"name": "google.com.",
		},
	})
	if len(alerts) != 0 {
		t.Fatal("expected no alert for benign query")
	}

	// Malicious query — should trigger
	e.Analyze(models.NetworkEvent{
		ID:   "2",
		Type: models.EventDNS,
		Metadata: map[string]any{
			"name": "data.evil.com.",
		},
	})
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Type != models.EventAlert {
		t.Fatalf("expected THREAT_ALERT, got %s", alerts[0].Type)
	}
}

func TestEngine_SkipsAlertEvents(t *testing.T) {
	var alerts []models.NetworkEvent
	e := newTestEngine()
	e.onAlert = func(ev models.NetworkEvent) { alerts = append(alerts, ev) }

	// Alert events should be skipped (no feedback loop)
	e.Analyze(models.NetworkEvent{
		ID:   "3",
		Type: models.EventAlert,
		Metadata: map[string]any{
			"name": "evil.com.",
		},
	})
	if len(alerts) != 0 {
		t.Fatal("alert events should be skipped to prevent feedback loops")
	}
}

func TestEngine_DisabledSkipsAnalysis(t *testing.T) {
	cfg := &config.CorrelationConfig{Enabled: false}
	e := &Engine{cfg: cfg, logger: testLogger(), rules: DefaultRules()}
	var alerts []models.NetworkEvent
	e.onAlert = func(ev models.NetworkEvent) { alerts = append(alerts, ev) }

	e.Analyze(models.NetworkEvent{
		ID:   "4",
		Type: models.EventDNS,
		Metadata: map[string]any{
			"name": "evil.com.",
		},
	})
	if len(alerts) != 0 {
		t.Fatal("disabled engine should not produce alerts")
	}
}

func TestEngine_NonDNSEventIgnored(t *testing.T) {
	var alerts []models.NetworkEvent
	e := newTestEngine()
	e.onAlert = func(ev models.NetworkEvent) { alerts = append(alerts, ev) }

	e.Analyze(models.NetworkEvent{
		ID:        "5",
		Type:      models.EventARP,
		Timestamp: time.Now(),
	})
	if len(alerts) != 0 {
		t.Fatal("non-DNS event should not trigger DNS exfiltration rule")
	}
}

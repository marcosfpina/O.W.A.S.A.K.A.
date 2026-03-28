package stream

import (
	"testing"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
)

func newTestProcessor() *Processor {
	cfg := &config.StreamConfig{
		Enabled:              true,
		BufferSize:           100,
		FlushIntervalSeconds: 60,
		Workers:              2,
	}
	return NewProcessor(cfg, nil)
}

func TestProcessor_EnrichAddsWindowStats(t *testing.T) {
	p := newTestProcessor()

	ev := models.NetworkEvent{
		ID:        "1",
		Type:      models.EventDNS,
		Source:    "192.168.1.10",
		Timestamp: time.Now(),
	}

	enriched := p.Enrich(ev)

	if enriched.Metadata == nil {
		t.Fatal("expected metadata to be populated")
	}
	if _, ok := enriched.Metadata["stream.count_1m"]; !ok {
		t.Fatal("expected stream.count_1m in metadata")
	}
	if _, ok := enriched.Metadata["stream.rate_1m"]; !ok {
		t.Fatal("expected stream.rate_1m in metadata")
	}
}

func TestProcessor_BufferCapacity(t *testing.T) {
	p := newTestProcessor()

	// Push 150 events into buffer of size 100
	for i := 0; i < 150; i++ {
		p.Enrich(models.NetworkEvent{
			ID:        "ev",
			Type:      models.EventARP,
			Source:    "10.0.0.1",
			Timestamp: time.Now(),
		})
	}

	stats := p.Stats()
	if stats.BufferedEvents != 100 {
		t.Fatalf("expected buffer capped at 100, got %d", stats.BufferedEvents)
	}
}

func TestProcessor_EmptySourceSkipsWindowing(t *testing.T) {
	p := newTestProcessor()

	ev := models.NetworkEvent{
		ID:        "1",
		Type:      models.EventDNS,
		Source:    "",
		Timestamp: time.Now(),
	}

	enriched := p.Enrich(ev)

	// No source → no window stats injected
	if _, ok := enriched.Metadata["stream.count_1m"]; ok {
		t.Fatal("expected no stream stats for empty source")
	}
}

func TestProcessor_WindowStatsForIP(t *testing.T) {
	p := newTestProcessor()

	for i := 0; i < 5; i++ {
		p.Enrich(models.NetworkEvent{
			ID:        "ev",
			Type:      models.EventDNS,
			Source:    "10.0.0.5",
			Timestamp: time.Now(),
		})
	}

	ws := p.WindowStatsForIP("10.0.0.5")
	if ws.Count1m != 5 {
		t.Fatalf("expected 5 events in 1m window, got %d", ws.Count1m)
	}
}

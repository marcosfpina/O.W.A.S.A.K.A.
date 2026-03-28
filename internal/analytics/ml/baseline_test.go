package ml

import (
	"testing"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
)

func TestBaseline_RecordAndFeatureVector(t *testing.T) {
	b := NewBaseline(0) // 0 → defaults to 7 days

	ev := models.NetworkEvent{
		Type:        models.EventDNS,
		Source:      "192.168.1.10",
		Destination: "8.8.8.8",
		Timestamp:   time.Now(),
	}

	b.Record(ev)
	b.Record(ev)

	features := b.FeatureVector("192.168.1.10", time.Now().Hour())
	if len(features) != 4 {
		t.Fatalf("expected 4 features, got %d", len(features))
	}

	// Event rate should be > 0
	if features[0] <= 0 {
		t.Fatal("event rate should be positive")
	}
}

func TestBaseline_EmptySourceIgnored(t *testing.T) {
	b := NewBaseline(1)

	b.Record(models.NetworkEvent{
		Type:      models.EventDNS,
		Source:    "",
		Timestamp: time.Now(),
	})

	features := b.FeatureVector("", time.Now().Hour())
	// No profile for empty source → zero vector
	for i, f := range features {
		if f != 0 {
			t.Fatalf("feature[%d] should be 0 for empty source, got %f", i, f)
		}
	}
}

func TestBaseline_UnknownHostReturnsZeros(t *testing.T) {
	b := NewBaseline(1)

	features := b.FeatureVector("unknown.host", 12)
	for i, f := range features {
		if f != 0 {
			t.Fatalf("feature[%d] should be 0 for unknown host, got %f", i, f)
		}
	}
}

func TestBaseline_IsLearning(t *testing.T) {
	// 7-day window — should be learning immediately
	b := NewBaseline(7)
	if !b.IsLearning() {
		t.Fatal("should be in learning phase initially")
	}

	// 0-second window (hack: set started in the past)
	b2 := NewBaseline(0)
	b2.started = time.Now().Add(-8 * 24 * time.Hour) // 8 days ago
	if b2.IsLearning() {
		t.Fatal("should NOT be learning after window elapsed")
	}
}

func TestBaseline_TrainingData(t *testing.T) {
	b := NewBaseline(1)

	// Record events for 3 different hosts
	for _, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		b.Record(models.NetworkEvent{
			Type:        models.EventDNS,
			Source:      ip,
			Destination: "8.8.8.8",
			Timestamp:   time.Now(),
		})
	}

	data := b.TrainingData()
	if len(data) != 3 {
		t.Fatalf("expected 3 training samples (one per host), got %d", len(data))
	}
	for i, row := range data {
		if len(row) != 4 {
			t.Fatalf("training sample %d has %d features, expected 4", i, len(row))
		}
	}
}

func TestBaseline_TypeEntropy(t *testing.T) {
	b := NewBaseline(1)

	// Single event type → entropy should be 0
	b.Record(models.NetworkEvent{
		Type:      models.EventDNS,
		Source:    "10.0.0.1",
		Timestamp: time.Now(),
	})

	features := b.FeatureVector("10.0.0.1", time.Now().Hour())
	entropy := features[3]
	if entropy != 0 {
		t.Fatalf("entropy for single type should be 0, got %f", entropy)
	}

	// Add different event types → entropy should increase
	b.Record(models.NetworkEvent{
		Type:      models.EventARP,
		Source:    "10.0.0.1",
		Timestamp: time.Now(),
	})
	b.Record(models.NetworkEvent{
		Type:      models.EventPortScan,
		Source:    "10.0.0.1",
		Timestamp: time.Now(),
	})

	features2 := b.FeatureVector("10.0.0.1", time.Now().Hour())
	if features2[3] <= 0 {
		t.Fatal("entropy should be positive with multiple event types")
	}
}

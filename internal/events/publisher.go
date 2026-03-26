package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Publisher wraps a NATS connection for publishing Spectre-schema events.
type Publisher struct {
	nc *nats.Conn
}

// Connect dials the NATS server at the given URL and returns a Publisher.
func Connect(natsURL string) (*Publisher, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("nats connect %s: %w", natsURL, err)
	}
	return &Publisher{nc: nc}, nil
}

// Close drains and closes the underlying NATS connection.
func (p *Publisher) Close() {
	if p.nc != nil {
		_ = p.nc.Drain()
	}
}

// Event mirrors the Spectre Event schema for JSON serialisation.
type Event struct {
	EventID       string         `json:"event_id"`
	EventType     string         `json:"event_type"`
	Timestamp     time.Time      `json:"timestamp"`
	SourceService string         `json:"source_service"`
	CorrelationID string         `json:"correlation_id"`
	Payload       map[string]any `json:"payload"`
}

// Publish serialises e and publishes it on the given NATS subject.
func (p *Publisher) Publish(subject string, e Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.nc.Publish(subject, data)
}

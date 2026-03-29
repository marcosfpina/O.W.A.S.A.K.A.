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
// The connection is configured for infinite reconnect so owasaka survives
// transient NATS restarts without losing event publishing capability.
func Connect(natsURL string) (*Publisher, error) {
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			fmt.Printf("[owasaka/publisher] NATS disconnected: %v\n", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("[owasaka/publisher] NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			fmt.Println("[owasaka/publisher] NATS connection closed")
		}),
	)
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
// Returns an error if the connection is not currently established; the
// caller should treat this as a transient failure — the connection will
// reconnect automatically and subsequent publishes will succeed.
func (p *Publisher) Publish(subject string, e Event) error {
	if !p.nc.IsConnected() {
		return fmt.Errorf("nats not connected (status: %s), event dropped: %s", p.nc.Status(), subject)
	}
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.nc.Publish(subject, data)
}

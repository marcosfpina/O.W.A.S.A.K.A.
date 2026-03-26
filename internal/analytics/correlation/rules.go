package correlation

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
)

// Rule represents an abstract threat detection signature (heuristic or Sigma-based)
type Rule interface {
	Name() string
	Evaluate(event models.NetworkEvent) *models.NetworkEvent
}

// DefaultRules populates the exact Zero-Trust rules for out-of-the-box alerting
func DefaultRules() []Rule {
	return []Rule{
		&DNSExfiltrationRule{},
	}
}

// DNSExfiltrationRule catches queries routed to malicious or data-hoarding TLDs
type DNSExfiltrationRule struct{}

// Name returns the formal threat signature label
func (r *DNSExfiltrationRule) Name() string {
	return "DNS_MALICIOUS_DOMAIN"
}

// Evaluate determines if the event violates the heuristics
func (r *DNSExfiltrationRule) Evaluate(e models.NetworkEvent) *models.NetworkEvent {
	if e.Type != models.EventDNS {
		return nil
	}
	
	name, ok := e.Metadata["name"].(string)
	if !ok {
		return nil
	}
	
	// Fast hardware string check bypassing heavy regex allocations for MVP
	if strings.Contains(name, "evil.com") || strings.Contains(name, "telemetry") {
		return &models.NetworkEvent{
			ID:          uuid.NewString(),
			Type:        models.EventAlert,
			Timestamp:   time.Now().UTC(),
			Source:      "CorrelationEngine",
			Destination: e.Source,
			Metadata: map[string]any{
				"rule":     r.Name(),
				"severity": "CRITICAL",
				"reason":   "DNS query targeting flagged malicious or telemetry domain",
				"target":   name,
			},
		}
	}
	
	return nil
}

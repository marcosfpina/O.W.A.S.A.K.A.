package dns

import (
	"fmt"
	"time"

	"github.com/miekg/dns"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// Resolver handles DNS queries
type Resolver struct {
	cfg    *config.DNSConfig
	logger *logging.Logger
	client *dns.Client
}

// NewResolver creates a new DNS resolver
func NewResolver(cfg *config.DNSConfig, logger *logging.Logger) *Resolver {
	return &Resolver{
		cfg:    cfg,
		logger: logger,
		client: &dns.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// ServeDNS handles incoming DNS requests
func (r *Resolver) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	// Prepare response
	msg := new(dns.Msg)
	msg.SetReply(req)
	msg.Compress = false
	msg.Authoritative = true

	// Handle each question in the request
	for _, q := range req.Question {
		// Log the query
		if r.cfg.LogQueries {
			r.logger.Infow("DNS Query",
				"src", w.RemoteAddr().String(),
				"type", dns.TypeToString[q.Qtype],
				"name", q.Name,
			)
		}

		// Forward to upstream
		// TODO: Implement caching and threat detection logic here
		resp, err := r.forward(req)
		if err != nil {
			r.logger.Errorw("Upstream propagation failed",
				"error", err,
				"name", q.Name,
			)
			dns.HandleFailed(w, req)
			return
		}

		if resp != nil {
			msg.Answer = append(msg.Answer, resp.Answer...)
			msg.Ns = append(msg.Ns, resp.Ns...)
			msg.Extra = append(msg.Extra, resp.Extra...)
		}
	}

	// Send response
	if err := w.WriteMsg(msg); err != nil {
		r.logger.Errorw("Failed to write DNS response", "error", err)
	}
}

// forward forwards a DNS request to an upstream server
func (r *Resolver) forward(req *dns.Msg) (*dns.Msg, error) {
	// Simple Round-Robin or first available upstream
	// For MVP, just use the first configured upstream
	if len(r.cfg.UpstreamServers) == 0 {
		return nil, fmt.Errorf("no upstream servers configured")
	}

	upstream := r.cfg.UpstreamServers[0]
	resp, _, err := r.client.Exchange(req, upstream)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

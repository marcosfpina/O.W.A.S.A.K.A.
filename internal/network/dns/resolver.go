package dns

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/miekg/dns"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/metrics"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

type cacheEntry struct {
	msg     *dns.Msg
	expires time.Time
}

// Resolver handles DNS queries with caching
type Resolver struct {
	cfg      *config.DNSConfig
	logger   *logging.Logger
	client   *dns.Client
	pipeline *events.Pipeline

	mu    sync.RWMutex
	cache map[string]*cacheEntry
}

// NewResolver creates a new DNS resolver
func NewResolver(cfg *config.DNSConfig, logger *logging.Logger, pipeline *events.Pipeline) *Resolver {
	cacheSize := cfg.CacheSize
	if cacheSize <= 0 {
		cacheSize = 10000
	}
	return &Resolver{
		cfg:    cfg,
		logger: logger,
		client: &dns.Client{
			Timeout: 2 * time.Second,
		},
		pipeline: pipeline,
		cache:    make(map[string]*cacheEntry, cacheSize),
	}
}

// StartCacheEvictor launches a background goroutine that removes expired entries.
func (r *Resolver) StartCacheEvictor(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.evictExpired()
			}
		}
	}()
}

func (r *Resolver) evictExpired() {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, entry := range r.cache {
		if now.After(entry.expires) {
			delete(r.cache, k)
		}
	}
}

func cacheKey(name string, qtype uint16) string {
	return fmt.Sprintf("%s:%d", name, qtype)
}

func (r *Resolver) cacheLookup(name string, qtype uint16) *dns.Msg {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.cache[cacheKey(name, qtype)]
	if !ok || time.Now().After(entry.expires) {
		return nil
	}
	return entry.msg.Copy()
}

func (r *Resolver) cacheStore(name string, qtype uint16, msg *dns.Msg) {
	ttl := r.cfg.CacheTTLSeconds
	if ttl <= 0 {
		ttl = 300
	}
	maxSize := r.cfg.CacheSize
	if maxSize <= 0 {
		maxSize = 10000
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.cache) >= maxSize {
		return // simple overflow protection — skip insertion
	}
	r.cache[cacheKey(name, qtype)] = &cacheEntry{
		msg:     msg.Copy(),
		expires: time.Now().Add(time.Duration(ttl) * time.Second),
	}
}

// ServeDNS handles incoming DNS requests
func (r *Resolver) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	if len(req.Question) > 0 {
		qtype := dns.TypeToString[req.Question[0].Qtype]
		if qtype == "" {
			qtype = "other"
		}
		metrics.DNSQueriesTotal.WithLabelValues(qtype).Inc()
	}

	msg := new(dns.Msg)
	msg.SetReply(req)
	msg.Compress = false
	msg.Authoritative = true

	for _, q := range req.Question {
		if r.cfg.LogQueries {
			r.logger.Infow("DNS Query",
				"src", w.RemoteAddr().String(),
				"type", dns.TypeToString[q.Qtype],
				"name", q.Name,
			)
		}

		if r.pipeline != nil {
			evt := models.NetworkEvent{
				ID:          uuid.NewString(),
				Type:        models.EventDNS,
				Timestamp:   time.Now().UTC(),
				Source:      w.RemoteAddr().String(),
				Destination: "SIEM_DNS",
				Metadata: map[string]any{
					"type": dns.TypeToString[q.Qtype],
					"name": q.Name,
				},
			}
			r.pipeline.PushNetworkEvent(evt)
		}

		// Check cache first
		if cached := r.cacheLookup(q.Name, q.Qtype); cached != nil {
			msg.Answer = append(msg.Answer, cached.Answer...)
			msg.Ns = append(msg.Ns, cached.Ns...)
			msg.Extra = append(msg.Extra, cached.Extra...)
			continue
		}

		// Forward to upstream
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
			r.cacheStore(q.Name, q.Qtype, resp)
			msg.Answer = append(msg.Answer, resp.Answer...)
			msg.Ns = append(msg.Ns, resp.Ns...)
			msg.Extra = append(msg.Extra, resp.Extra...)
		}
	}

	if err := w.WriteMsg(msg); err != nil {
		r.logger.Errorw("Failed to write DNS response", "error", err)
	}
}

// forward forwards a DNS request to an upstream server
func (r *Resolver) forward(req *dns.Msg) (*dns.Msg, error) {
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

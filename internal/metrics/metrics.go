// Package metrics defines Prometheus metrics for owasaka.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// EventsPublished counts NATS events emitted by owasaka, labelled by subject.
	EventsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "owasaka_events_published_total",
			Help: "Total NATS events published by owasaka, by Spectre subject.",
		},
		[]string{"subject"},
	)

	// AssetsDiscovered counts unique assets discovered since process start.
	AssetsDiscovered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "owasaka_assets_discovered_total",
			Help: "Total network assets discovered (all discovery methods).",
		},
	)

	// DNSQueriesTotal counts DNS queries observed.
	DNSQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "owasaka_dns_queries_total",
			Help: "Total DNS queries observed by the DNS monitor.",
		},
		[]string{"type"},
	)

	// HTTPRequestsTotal counts HTTP requests handled by the API server.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "owasaka_http_requests_total",
			Help: "Total HTTP requests handled by the owasaka API server.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration records API request latency.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "owasaka_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// ThreatEventsTotal counts threat/anomaly events detected.
	ThreatEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "owasaka_threat_events_total",
			Help: "Total threat/anomaly events detected by the correlation engine.",
		},
		[]string{"severity"},
	)
)

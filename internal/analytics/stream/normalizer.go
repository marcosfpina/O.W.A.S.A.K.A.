package stream

import (
	"net"
	"strings"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
)

// normalizeIP strips port info and returns a canonical IPv4/IPv6 string.
// Returns empty string for broadcasts, multicasts, and unparseable addresses.
func normalizeIP(raw string) string {
	if raw == "" {
		return ""
	}
	// Strip port if present
	host, _, err := net.SplitHostPort(raw)
	if err != nil {
		host = raw
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return raw // return as-is (may be a hostname)
	}
	// Discard broadcasts and link-local multicast
	if ip.IsMulticast() || ip.IsLinkLocalMulticast() {
		return ""
	}
	return ip.String()
}

// Normalize returns a copy of the event with canonical source/destination IPs
// and a consistent EventType string.
func Normalize(e models.NetworkEvent) models.NetworkEvent {
	e.Source = normalizeIP(e.Source)
	e.Destination = normalizeIP(e.Destination)
	return e
}

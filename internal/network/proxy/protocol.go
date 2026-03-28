package proxy

import (
	"net/http"
	"strings"
)

// Protocol classifies observed traffic
type Protocol string

const (
	ProtoHTTP      Protocol = "HTTP"
	ProtoHTTPS     Protocol = "HTTPS"
	ProtoWebSocket Protocol = "WebSocket"
	ProtoGRPC      Protocol = "gRPC"
	ProtoUnknown   Protocol = "Unknown"
)

// DetectProtocol infers the application protocol from the request and target port.
func DetectProtocol(r *http.Request, port string) Protocol {
	// WebSocket upgrade
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return ProtoWebSocket
	}
	// gRPC uses content-type application/grpc
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
		return ProtoGRPC
	}
	// CONNECT method implies TLS tunnel
	if r.Method == http.MethodConnect {
		return ProtoHTTPS
	}
	// Port-based fallback
	switch port {
	case "443", "8443":
		return ProtoHTTPS
	default:
		return ProtoHTTP
	}
}

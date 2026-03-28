package proxy

import (
	"net/http"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/events"
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// interceptor captures HTTP request metadata and emits pipeline events.
type interceptor struct {
	pipeline *events.Pipeline
	logger   *logging.Logger
}

func newInterceptor(pipeline *events.Pipeline, logger *logging.Logger) *interceptor {
	return &interceptor{pipeline: pipeline, logger: logger}
}

// logRequest emits a PROXY NetworkEvent for a captured HTTP request.
func (i *interceptor) logRequest(r *http.Request, port string, dur time.Duration, statusCode int) {
	proto := DetectProtocol(r, port)
	meta := ExtractMetadata(r, proto)
	meta["duration_ms"] = dur.Milliseconds()
	if statusCode > 0 {
		meta["status_code"] = statusCode
	}

	clientIP := r.RemoteAddr
	targetHost := r.Host

	i.pipeline.PushNetworkEvent(models.NetworkEvent{
		Type:        models.EventProxy,
		Source:      clientIP,
		Destination: targetHost,
		Metadata:    meta,
		Timestamp:   time.Now(),
	})
}

// logTunnel emits a PROXY event for a CONNECT tunnel (no decryption).
func (i *interceptor) logTunnel(clientAddr, targetHost string) {
	i.pipeline.PushNetworkEvent(models.NetworkEvent{
		Type:        models.EventProxy,
		Source:      clientAddr,
		Destination: targetHost,
		Metadata: map[string]any{
			"method":   "CONNECT",
			"protocol": string(ProtoHTTPS),
			"tunnel":   true,
		},
		Timestamp: time.Now(),
	})
}

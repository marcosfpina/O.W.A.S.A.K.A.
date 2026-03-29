package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/metrics"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server encapsulates the HTTP and WebSocket API server
type Server struct {
	cfg        *config.ServerConfig
	logger     *logging.Logger
	httpServer *http.Server
	Hub        *WSHub
	mux        *http.ServeMux
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Instrument wraps a handler with Prometheus request metrics (exported for use in app.go).
func Instrument(path string, h http.HandlerFunc) http.HandlerFunc {
	return instrumentedHandler(path, h)
}

// instrumentedHandler wraps a handler with Prometheus request metrics.
func instrumentedHandler(path string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		h(rw, r)
		dur := time.Since(start).Seconds()
		status := strconv.Itoa(rw.status)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(dur)
	}
}

// NewServer builds a new API manager
func NewServer(cfg *config.ServerConfig, logger *logging.Logger) *Server {
	s := &Server{
		cfg:    cfg,
		logger: logger,
		Hub:    NewWSHub(logger),
		mux:    http.NewServeMux(),
	}
	s.mux.HandleFunc("/health", instrumentedHandler("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"online"}`))
	}))
	s.mux.Handle("/metrics", promhttp.Handler())
	return s
}

// RegisterHandler registers an HTTP handler before or after Start
func (s *Server) RegisterHandler(path string, handler http.HandlerFunc) {
	s.mux.HandleFunc(path, handler)
}

// Start opens the main port listeners
func (s *Server) Start(ctx context.Context) error {
	go s.Hub.Run()

	if s.cfg.WebSocket.Enabled {
		s.mux.HandleFunc(s.cfg.WebSocket.Path, s.Hub.Handler)
		s.logger.Infow("WebSocket endpoint registered", "path", s.cfg.WebSocket.Path)
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	s.logger.Infow("Starting Command Center API", "addr", addr)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Errorw("Command Center API failed", "error", err)
		}
	}()

	return nil
}

// Stop enforces an elegant drop of API traffic
func (s *Server) Stop() {
	s.logger.Info("Stopping Command Center API")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Errorw("Failed to gracefully shutdown API", "error", err)
	}
}

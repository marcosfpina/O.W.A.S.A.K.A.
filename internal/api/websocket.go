package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// WSHub maintains the set of active clients and broadcasts messages
type WSHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	logger     *logging.Logger
	mu         sync.Mutex // explicitly guard the broadcast sending
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow any origin for the MVP (Command Center runs locally in SvelteKit dev server)
		return true
	},
}

// NewWSHub constructs a new WebSocket Hub
func NewWSHub(logger *logging.Logger) *WSHub {
	return &WSHub{
		broadcast:  make(chan interface{}, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		clients:    make(map[*websocket.Conn]bool),
		logger:     logger,
	}
}

// Run starts the central hub loop
func (h *WSHub) Run() {
	h.logger.Info("Starting WebSocket Hub")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Infow("WebSocket client connected", "clients", len(h.clients))
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
				h.logger.Infow("WebSocket client disconnected", "clients", len(h.clients))
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			// Encode once to avoid encoding loops per client
			data, err := json.Marshal(message)
			if err != nil {
				h.logger.Errorw("Failed to marshal WS message", "error", err)
				h.mu.Unlock()
				continue
			}

			for client := range h.clients {
				client.SetWriteDeadline(time.Now().Add(10 * time.Second))
				err := client.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					h.logger.Errorw("WebSocket write error", "error", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// Broadcast pushes an arbitrary struct to all connected clients
func (h *WSHub) Broadcast(msg interface{}) {
	// Non-blocking send or drop if channel is full
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("WebSocket broadcast channel full, dropping message")
	}
}

// Handler handles websocket requests from the peer
func (h *WSHub) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Errorw("WebSocket upgrade failed", "error", err, "client", r.RemoteAddr)
		return
	}
	h.register <- conn

	// Pump read loop to keep connection alive and catch disconnects
	go func() {
		defer func() {
			h.unregister <- conn
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

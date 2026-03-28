package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// CDPClient communicates with Firefox via the Chrome DevTools Protocol.
// Firefox supports CDP on --remote-debugging-port.
type CDPClient struct {
	logger  *logging.Logger
	port    int
	conn    *websocket.Conn
	mu      sync.Mutex
	nextID  atomic.Int64
	pending map[int64]chan json.RawMessage
}

// NewCDPClient creates a CDP client targeting a given debug port.
func NewCDPClient(port int, logger *logging.Logger) *CDPClient {
	return &CDPClient{
		port:    port,
		logger:  logger,
		pending: make(map[int64]chan json.RawMessage),
	}
}

// Connect discovers the debug WebSocket endpoint and opens a connection.
func (c *CDPClient) Connect(ctx context.Context) error {
	// Firefox exposes /json/version with the wsDebuggerUrl
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", c.port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("CDP discovery failed (is Firefox running with --remote-debugging-port?): %w", err)
	}
	defer resp.Body.Close()

	var version struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return fmt.Errorf("CDP version parse failed: %w", err)
	}

	if version.WebSocketDebuggerURL == "" {
		return fmt.Errorf("no WebSocket debugger URL in CDP version response")
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, version.WebSocketDebuggerURL, nil)
	if err != nil {
		return fmt.Errorf("CDP websocket dial failed: %w", err)
	}

	c.conn = conn
	go c.readLoop()
	c.logger.Infow("CDP connected", "port", c.port, "ws", version.WebSocketDebuggerURL)
	return nil
}

// Close shuts down the CDP connection.
func (c *CDPClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

type cdpRequest struct {
	ID     int64          `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

type cdpResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Send sends a CDP command and waits for the response.
func (c *CDPClient) Send(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	ch := make(chan json.RawMessage, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	req := cdpRequest{ID: id, Method: method, Params: params}
	data, _ := json.Marshal(req)

	c.mu.Lock()
	err := c.conn.WriteMessage(websocket.TextMessage, data)
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("CDP write failed: %w", err)
	}

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("CDP command timeout: %s", method)
	}
}

func (c *CDPClient) readLoop() {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.Warnw("CDP read error", "error", err)
			return
		}

		var resp cdpResponse
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue
		}

		if resp.ID > 0 {
			c.mu.Lock()
			ch, ok := c.pending[resp.ID]
			c.mu.Unlock()
			if ok {
				if resp.Error != nil {
					ch <- json.RawMessage(fmt.Sprintf(`{"error":"%s"}`, resp.Error.Message))
				} else {
					ch <- resp.Result
				}
			}
		}
	}
}

// CaptureScreenshot takes a PNG screenshot of the current page.
func (c *CDPClient) CaptureScreenshot(ctx context.Context) ([]byte, error) {
	result, err := c.Send(ctx, "Page.captureScreenshot", map[string]any{
		"format": "png",
	})
	if err != nil {
		return nil, err
	}

	var screenshot struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &screenshot); err != nil {
		return nil, err
	}

	return []byte(screenshot.Data), nil // base64-encoded PNG
}

// GetNavigationHistory returns the browser's navigation entries.
func (c *CDPClient) GetNavigationHistory(ctx context.Context) (json.RawMessage, error) {
	return c.Send(ctx, "Page.getNavigationHistory", nil)
}

// EnableNetwork enables network domain events (for HAR-like capture).
func (c *CDPClient) EnableNetwork(ctx context.Context) error {
	_, err := c.Send(ctx, "Network.enable", nil)
	return err
}

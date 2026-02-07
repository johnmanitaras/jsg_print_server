package cloud

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jetsetgo/local-print-server/internal/config"
	"github.com/jetsetgo/local-print-server/internal/printer"
)

// Cloud WebSocket close codes
const (
	CloseReplaced      = 4000 // Newer connection from same server replaced this one
	CloseAuthFailure   = 4001 // Missing headers, invalid API key, or unknown server
	CloseLimitExceeded = 4002 // Too many connections (500 global, 50 per tenant)
	CloseWrongWorker   = 4003 // Hit API container instead of background worker (routing misconfigured)
)

// WSClient manages the WebSocket connection to the cloud server
type WSClient struct {
	config     *config.CloudConfig
	printerMgr *printer.Manager
	conn       *websocket.Conn
	mu         sync.Mutex

	// State
	connected    bool
	reconnecting bool
	lastError    error
	lastSeen     time.Time

	// Channels
	done chan struct{}
	send chan []byte

	// Callback for job tracking
	OnJobReceived  func(jobID, printerID string, dataSize int)
	OnJobCompleted func(jobID, status, errMsg string)

	// PrinterList returns printer configs for syncing to cloud
	PrinterList func() []map[string]interface{}

	// OnFallbackToPoll is called when WS determines it cannot connect
	// and the server should fall back to HTTP polling (e.g. close code 4003)
	OnFallbackToPoll func()
}

// IncomingMessage represents messages from the cloud
type IncomingMessage struct {
	Type      string                 `json:"type"`
	JobID     string                 `json:"job_id,omitempty"`
	PrinterID string                 `json:"printer_id,omitempty"`
	Priority  int                    `json:"priority,omitempty"`
	Data      string                 `json:"data,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// OutgoingMessage represents messages to the cloud
type OutgoingMessage struct {
	Type   string `json:"type"`
	JobID  string `json:"job_id,omitempty"`
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

// NewWSClient creates a new WebSocket client
func NewWSClient(cfg *config.CloudConfig, printerMgr *printer.Manager) *WSClient {
	return &WSClient{
		config:     cfg,
		printerMgr: printerMgr,
		done:       make(chan struct{}),
		send:       make(chan []byte, 10),
	}
}

// Start begins the WebSocket connection and reconnection loop
func (c *WSClient) Start() {
	go c.connectionLoop()
}

// Stop gracefully closes the WebSocket connection
func (c *WSClient) Stop() {
	close(c.done)
	c.mu.Lock()
	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
	}
	c.mu.Unlock()
}

// Status returns the current connection status
func (c *WSClient) Status() ConnectionStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	errStr := ""
	if c.lastError != nil {
		errStr = c.lastError.Error()
	}

	return ConnectionStatus{
		Connected:    c.connected,
		Reconnecting: c.reconnecting,
		LastError:    errStr,
		LastSeen:     c.lastSeen,
	}
}

// SyncPrinters forces a printer sync via HTTP
func (c *WSClient) SyncPrinters() {
	go c.syncPrinters()
}

func (c *WSClient) syncPrinters() {
	if c.PrinterList == nil {
		return
	}

	printers := c.PrinterList()
	url := fmt.Sprintf("%s/servers/%s/printers", c.config.Endpoint, c.config.ServerID)

	payload, _ := json.Marshal(map[string]interface{}{
		"printers": printers,
	})

	req, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("Failed to create printer sync request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.config.APIKey)
	if c.config.Tenant != "" {
		req.Header.Set("X-DB-Name", c.config.Tenant)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to sync printers: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Printer sync returned %d: %s", resp.StatusCode, string(body))
		return
	}

	log.Printf("Printers synced to cloud (%d printers)", len(printers))
}

// connectionLoop manages connection and reconnection
func (c *WSClient) connectionLoop() {
	delay := c.config.WSReconnectDelay

	for {
		select {
		case <-c.done:
			return
		default:
		}

		closeCode, err := c.connectAndRun()
		if err != nil {
			c.mu.Lock()
			c.connected = false
			c.reconnecting = true
			c.lastError = err
			c.mu.Unlock()

			// Handle specific close codes
			switch closeCode {
			case CloseAuthFailure:
				log.Printf("WebSocket auth failed (4001): %v. Check API key and tenant. Not reconnecting.", err)
				c.mu.Lock()
				c.reconnecting = false
				c.mu.Unlock()
				return

			case CloseWrongWorker:
				log.Printf("WebSocket routing error (4003): reverse proxy not configured. Falling back to HTTP polling.")
				c.mu.Lock()
				c.reconnecting = false
				c.mu.Unlock()
				if c.OnFallbackToPoll != nil {
					c.OnFallbackToPoll()
				}
				return

			case CloseLimitExceeded:
				delay = 30 * time.Second
				log.Printf("WebSocket connection limit exceeded (4002). Retrying in %v...", delay)

			case CloseReplaced:
				delay = c.config.WSReconnectDelay
				log.Printf("WebSocket connection replaced by newer connection (4000). Reconnecting in %v...", delay)

			default:
				log.Printf("WebSocket disconnected: %v. Reconnecting in %v...", err, delay)
			}

			select {
			case <-c.done:
				return
			case <-time.After(delay):
			}

			// Exponential backoff for generic errors
			if closeCode != CloseReplaced && closeCode != CloseLimitExceeded {
				delay = delay * 2
				if delay > c.config.WSMaxReconnect {
					delay = c.config.WSMaxReconnect
				}
			}
			continue
		}

		// Disconnected cleanly - reset delay and reconnect
		delay = c.config.WSReconnectDelay
	}
}

// connectAndRun establishes the connection and runs read/write loops.
// Returns the close code (0 if unknown) and error.
func (c *WSClient) connectAndRun() (int, error) {
	wsURL := c.config.WSEndpoint

	header := http.Header{}
	if c.config.APIKey != "" {
		header.Set("X-API-Key", c.config.APIKey)
	}
	if c.config.Tenant != "" {
		header.Set("X-DB-Name", c.config.Tenant)
	}

	log.Printf("Connecting to WebSocket: %s", wsURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusUnauthorized, http.StatusForbidden:
				return CloseAuthFailure, fmt.Errorf("authentication failed (HTTP %d)", resp.StatusCode)
			case http.StatusNotFound, http.StatusBadGateway, http.StatusServiceUnavailable:
				// Endpoint doesn't exist or routing not configured - same as 4003
				return CloseWrongWorker, fmt.Errorf("WebSocket endpoint not available (HTTP %d)", resp.StatusCode)
			}
		}
		return 0, fmt.Errorf("dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.reconnecting = false
	c.lastError = nil
	c.lastSeen = time.Now()
	c.mu.Unlock()

	log.Println("WebSocket connected")

	// Sync printers on connect (via HTTP)
	go c.syncPrinters()

	// Run read/write loops until disconnection
	closeCode := c.runConnection()

	c.mu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	log.Println("WebSocket disconnected")

	if closeCode != 0 {
		return closeCode, fmt.Errorf("connection closed with code %d", closeCode)
	}
	return 0, fmt.Errorf("connection lost")
}

// runConnection handles read/write on an established connection.
// Returns the WebSocket close code (0 if unknown).
func (c *WSClient) runConnection() int {
	closeCodeCh := make(chan int, 1)
	readDone := make(chan struct{})

	// Read loop
	go func() {
		defer close(readDone)
		for {
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()

			if conn == nil {
				return
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				code := extractCloseCode(err)
				closeCodeCh <- code
				return
			}

			c.mu.Lock()
			c.lastSeen = time.Now()
			c.mu.Unlock()

			c.handleMessage(message)
		}
	}()

	// Write loop (pings and outgoing messages)
	ticker := time.NewTicker(c.config.WSPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return 0

		case <-readDone:
			select {
			case code := <-closeCodeCh:
				return code
			default:
				return 0
			}

		case message := <-c.send:
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()

			if conn == nil {
				return 0
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return 0
			}

		case <-ticker.C:
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()

			if conn == nil {
				return 0
			}

			// Send application-level ping
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			pingMsg, _ := json.Marshal(OutgoingMessage{Type: "ping"})
			if err := conn.WriteMessage(websocket.TextMessage, pingMsg); err != nil {
				log.Printf("WebSocket ping error: %v", err)
				return 0
			}
		}
	}
}

// extractCloseCode extracts the WebSocket close code from an error
func extractCloseCode(err error) int {
	if closeErr, ok := err.(*websocket.CloseError); ok {
		return closeErr.Code
	}
	return 0
}

// handleMessage processes incoming WebSocket messages
func (c *WSClient) handleMessage(data []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to parse WebSocket message: %v", err)
		return
	}

	switch msg.Type {
	case "job":
		go c.handleJob(msg)
	case "pong":
		// Heartbeat response - connection is alive
	default:
		log.Printf("Unknown WebSocket message type: %s", msg.Type)
	}
}

// handleJob processes an incoming print job
func (c *WSClient) handleJob(msg IncomingMessage) {
	log.Printf("Received print job via WebSocket: %s for printer: %s", msg.JobID, msg.PrinterID)

	if msg.Data == "" {
		log.Printf("Skipping job %s: no data", msg.JobID)
		c.sendStatus(msg.JobID, "failed", "no print data")
		if c.OnJobCompleted != nil {
			c.OnJobCompleted(msg.JobID, "failed", "no print data")
		}
		return
	}

	// Decode base64 ESC/POS data
	escposData, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		log.Printf("Failed to decode job data: %v", err)
		c.sendStatus(msg.JobID, "failed", fmt.Sprintf("decode error: %v", err))
		if c.OnJobCompleted != nil {
			c.OnJobCompleted(msg.JobID, "failed", fmt.Sprintf("decode error: %v", err))
		}
		return
	}

	if c.OnJobReceived != nil {
		c.OnJobReceived(msg.JobID, msg.PrinterID, len(escposData))
	}

	// Report "printing" status
	c.sendStatus(msg.JobID, "printing", "")

	// Send to printer
	err = c.printerMgr.Print(msg.PrinterID, escposData)
	if err != nil {
		log.Printf("Print failed: %v", err)
		c.sendStatus(msg.JobID, "failed", err.Error())
		if c.OnJobCompleted != nil {
			c.OnJobCompleted(msg.JobID, "failed", err.Error())
		}
		return
	}

	log.Printf("Print job %s completed successfully", msg.JobID)
	c.sendStatus(msg.JobID, "completed", "")
	if c.OnJobCompleted != nil {
		c.OnJobCompleted(msg.JobID, "completed", "")
	}
}

// sendStatus sends a job status update via WebSocket
func (c *WSClient) sendStatus(jobID, status, errMsg string) {
	msg := OutgoingMessage{
		Type:   "status",
		JobID:  jobID,
		Status: status,
	}
	if errMsg != "" {
		msg.Error = errMsg
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal status message: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		log.Println("WebSocket send channel full, dropping status message")
	}
}

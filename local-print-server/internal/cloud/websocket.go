package cloud

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jetsetgo/local-print-server/internal/config"
	"github.com/jetsetgo/local-print-server/internal/printer"
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

	// Channels
	done     chan struct{}
	send     chan []byte
	statusCh chan ConnectionStatus
}

// ConnectionStatus represents the WebSocket connection status
type ConnectionStatus struct {
	Connected    bool
	Reconnecting bool
	LastError    string
	LastSeen     time.Time
}

// Message types from cloud
type IncomingMessage struct {
	Type      string `json:"type"`
	JobID     string `json:"job_id,omitempty"`
	PrinterID string `json:"printer_id,omitempty"`
	Data      string `json:"data,omitempty"` // base64 encoded ESC/POS
}

// Message types to cloud
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
		statusCh:   make(chan ConnectionStatus, 1),
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
	}
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

		err := c.connect()
		if err != nil {
			c.mu.Lock()
			c.connected = false
			c.reconnecting = true
			c.lastError = err
			c.mu.Unlock()

			log.Printf("WebSocket connection failed: %v. Reconnecting in %v...", err, delay)

			select {
			case <-c.done:
				return
			case <-time.After(delay):
			}

			// Exponential backoff
			delay = delay * 2
			if delay > c.config.WSMaxReconnect {
				delay = c.config.WSMaxReconnect
			}
			continue
		}

		// Connected successfully, reset delay
		delay = c.config.WSReconnectDelay

		// Run read/write loops until disconnection
		c.runConnection()
	}
}

// connect establishes the WebSocket connection
func (c *WSClient) connect() error {
	// Build WebSocket URL with server ID
	wsURL := c.config.WSEndpoint
	if c.config.ServerID != "" {
		wsURL = strings.Replace(wsURL, "{server_id}", c.config.ServerID, 1)
	}

	// Add API key and tenant to headers
	header := http.Header{}
	if c.config.APIKey != "" {
		header.Set("X-API-Key", c.config.APIKey)
	}
	if c.config.Tenant != "" {
		header.Set("X-DB-Name", c.config.Tenant)
	}

	log.Printf("Connecting to WebSocket: %s", wsURL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.reconnecting = false
	c.lastError = nil
	c.mu.Unlock()

	log.Println("WebSocket connected successfully")
	return nil
}

// runConnection handles read/write on an established connection
func (c *WSClient) runConnection() {
	var wg sync.WaitGroup
	wg.Add(2)

	// Read loop
	go func() {
		defer wg.Done()
		c.readLoop()
	}()

	// Write loop (handles pings and outgoing messages)
	go func() {
		defer wg.Done()
		c.writeLoop()
	}()

	wg.Wait()

	c.mu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
}

// readLoop reads incoming messages from the WebSocket
func (c *WSClient) readLoop() {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		c.handleMessage(message)
	}
}

// writeLoop handles outgoing messages and ping/pong
func (c *WSClient) writeLoop() {
	ticker := time.NewTicker(c.config.WSPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return

		case message := <-c.send:
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()

			if conn == nil {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()

			if conn == nil {
				return
			}

			// Send ping
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			pingMsg := OutgoingMessage{Type: "ping"}
			data, _ := json.Marshal(pingMsg)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WebSocket ping error: %v", err)
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *WSClient) handleMessage(data []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to parse message: %v", err)
		return
	}

	switch msg.Type {
	case "job":
		c.handleJob(msg)
	case "pong":
		// Heartbeat response, nothing to do
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handleJob processes an incoming print job
func (c *WSClient) handleJob(msg IncomingMessage) {
	log.Printf("Received print job: %s for printer: %s", msg.JobID, msg.PrinterID)

	// Decode base64 ESC/POS data
	escposData, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		log.Printf("Failed to decode job data: %v", err)
		c.sendStatus(msg.JobID, "failed", fmt.Sprintf("decode error: %v", err))
		return
	}

	// Send to printer
	err = c.printerMgr.Print(msg.PrinterID, escposData)
	if err != nil {
		log.Printf("Print failed: %v", err)
		c.sendStatus(msg.JobID, "failed", err.Error())
		return
	}

	log.Printf("Print job %s completed successfully", msg.JobID)
	c.sendStatus(msg.JobID, "completed", "")
}

// sendStatus sends a job status update to the cloud
func (c *WSClient) sendStatus(jobID, status, errMsg string) {
	msg := OutgoingMessage{
		Type:   "status",
		JobID:  jobID,
		Status: status,
		Error:  errMsg,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal status message: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		log.Println("Send channel full, dropping status message")
	}
}

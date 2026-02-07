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

	"github.com/jetsetgo/local-print-server/internal/config"
	"github.com/jetsetgo/local-print-server/internal/printer"
)

// PollClient polls the cloud for pending print jobs and sends heartbeats
type PollClient struct {
	config     *config.CloudConfig
	printerMgr *printer.Manager
	client     *http.Client
	mu         sync.Mutex

	connected bool
	lastError error
	lastSeen  time.Time

	done chan struct{}

	// Callback for job tracking
	OnJobReceived  func(jobID, printerID string, dataSize int)
	OnJobCompleted func(jobID, status, errMsg string)

	// PrinterStatuses returns current printer statuses for heartbeat
	PrinterStatuses func() map[string]string

	// PrinterList returns printer configs for syncing to cloud
	PrinterList func() []map[string]interface{}

	printersSynced bool
}

// NewPollClient creates a new polling client
func NewPollClient(cfg *config.CloudConfig, printerMgr *printer.Manager) *PollClient {
	return &PollClient{
		config:     cfg,
		printerMgr: printerMgr,
		client:     &http.Client{Timeout: 15 * time.Second},
		done:       make(chan struct{}),
	}
}

// Start begins the polling loop
func (p *PollClient) Start() {
	go p.pollLoop()
}

// Stop stops the polling loop
func (p *PollClient) Stop() {
	close(p.done)
}

// Status returns the current connection status
func (p *PollClient) Status() ConnectionStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	errStr := ""
	if p.lastError != nil {
		errStr = p.lastError.Error()
	}

	return ConnectionStatus{
		Connected:    p.connected,
		Reconnecting: false,
		LastError:    errStr,
		LastSeen:     p.lastSeen,
	}
}

func (p *PollClient) pollLoop() {
	// Sync printers, send heartbeat, and poll immediately on start
	p.syncPrinters()
	p.sendHeartbeat()
	p.poll()

	pollTicker := time.NewTicker(p.config.PollInterval)
	defer pollTicker.Stop()

	// Heartbeat every 30 seconds
	heartbeatInterval := 30 * time.Second
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-pollTicker.C:
			p.poll()
		case <-heartbeatTicker.C:
			p.sendHeartbeat()
		}
	}
}

type pollJob struct {
	JobID     string `json:"job_id"`
	PrinterID string `json:"printer_id"`
	Data      string `json:"data"`
	Status    string `json:"status"`
}

type pollJobResponse struct {
	Jobs []pollJob `json:"jobs"`
}

func (p *PollClient) poll() {
	url := fmt.Sprintf("%s/servers/%s/jobs", p.config.Endpoint, p.config.ServerID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		p.setError(err)
		return
	}

	req.Header.Set("X-API-Key", p.config.APIKey)
	if p.config.Tenant != "" {
		req.Header.Set("X-DB-Name", p.config.Tenant)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.setError(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.setError(fmt.Errorf("poll returned %d: %s", resp.StatusCode, string(body)))
		return
	}

	// Successfully reached cloud
	p.mu.Lock()
	p.connected = true
	p.lastError = nil
	p.lastSeen = time.Now()
	p.mu.Unlock()

	var result pollJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to parse poll response: %v", err)
		return
	}

	for _, job := range result.Jobs {
		if job.Status != "pending" {
			continue
		}
		go p.processJob(job.JobID, job.PrinterID, job.Data)
	}
}

func (p *PollClient) processJob(jobID, printerID, data string) {
	log.Printf("Received print job via polling: %s for printer: %s", jobID, printerID)

	escposData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Printf("Failed to decode job data: %v", err)
		p.reportStatus(jobID, "failed", fmt.Sprintf("decode error: %v", err))
		return
	}

	if p.OnJobReceived != nil {
		p.OnJobReceived(jobID, printerID, len(escposData))
	}

	err = p.printerMgr.Print(printerID, escposData)
	if err != nil {
		log.Printf("Print failed: %v", err)
		p.reportStatus(jobID, "failed", err.Error())
		if p.OnJobCompleted != nil {
			p.OnJobCompleted(jobID, "failed", err.Error())
		}
		return
	}

	log.Printf("Print job %s completed successfully", jobID)
	p.reportStatus(jobID, "completed", "")
	if p.OnJobCompleted != nil {
		p.OnJobCompleted(jobID, "completed", "")
	}
}

func (p *PollClient) reportStatus(jobID, status, errMsg string) {
	url := fmt.Sprintf("%s/servers/%s/jobs/%s", p.config.Endpoint, p.config.ServerID, jobID)

	payload, _ := json.Marshal(map[string]string{
		"status": status,
		"error":  errMsg,
	})

	req, err := http.NewRequest("PATCH", url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("Failed to create status request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.config.APIKey)
	if p.config.Tenant != "" {
		req.Header.Set("X-DB-Name", p.config.Tenant)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("Failed to report job status: %v", err)
		return
	}
	resp.Body.Close()
}

func (p *PollClient) sendHeartbeat() {
	url := fmt.Sprintf("%s/servers/%s/heartbeat", p.config.Endpoint, p.config.ServerID)

	printers := map[string]string{}
	if p.PrinterStatuses != nil {
		printers = p.PrinterStatuses()
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"printers": printers,
	})

	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("Failed to create heartbeat request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.config.APIKey)
	if p.config.Tenant != "" {
		req.Header.Set("X-DB-Name", p.config.Tenant)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.setError(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Heartbeat returned %d: %s", resp.StatusCode, string(body))
		return
	}

	// Heartbeat success means we're connected
	p.mu.Lock()
	p.connected = true
	p.lastError = nil
	p.lastSeen = time.Now()
	p.mu.Unlock()
}

// SyncPrinters forces a printer sync on next opportunity
func (p *PollClient) SyncPrinters() {
	p.mu.Lock()
	p.printersSynced = false
	p.mu.Unlock()
	go p.syncPrinters()
}

func (p *PollClient) syncPrinters() {
	if p.PrinterList == nil {
		return
	}

	printers := p.PrinterList()
	url := fmt.Sprintf("%s/servers/%s/printers", p.config.Endpoint, p.config.ServerID)

	payload, _ := json.Marshal(map[string]interface{}{
		"printers": printers,
	})

	req, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("Failed to create printer sync request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.config.APIKey)
	if p.config.Tenant != "" {
		req.Header.Set("X-DB-Name", p.config.Tenant)
	}

	resp, err := p.client.Do(req)
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

	p.mu.Lock()
	p.printersSynced = true
	p.mu.Unlock()

	log.Printf("Printers synced to cloud (%d printers)", len(printers))
}

func (p *PollClient) setError(err error) {
	p.mu.Lock()
	p.connected = false
	p.lastError = err
	p.mu.Unlock()
}

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jetsetgo/local-print-server/internal/cloud"
	"github.com/jetsetgo/local-print-server/internal/config"
	"github.com/jetsetgo/local-print-server/internal/printer"
)

// Server represents the HTTP server
type Server struct {
	config         *config.Config
	configMu       sync.RWMutex
	printerManager *printer.Manager
	wsClient       *cloud.WSClient
	pollClient     *cloud.PollClient
	mux            *http.ServeMux
	logBuffer      *LogBuffer
	jobBuffer      *JobBuffer
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, logBuf *LogBuffer, jobBuf *JobBuffer) *Server {
	printerMgr := printer.NewManager()

	s := &Server{
		config:         cfg,
		printerManager: printerMgr,
		mux:            http.NewServeMux(),
		logBuffer:      logBuf,
		jobBuffer:      jobBuf,
	}

	// Load printers from configuration
	for _, p := range cfg.Printers {
		switch p.Type {
		case "network":
			np := printer.NewNetworkPrinter(p.ID, p.Name, p.Address, p.Port)
			s.printerManager.AddPrinter(np)
		case "usb":
			// TODO: Add USB printer support
		}
	}

	// Create cloud client if configured
	if cfg.Cloud.ServerID != "" && cfg.Cloud.APIKey != "" {
		if cfg.Cloud.UseWebSocket {
			s.wsClient = cloud.NewWSClient(&cfg.Cloud, printerMgr)
		} else {
			s.pollClient = cloud.NewPollClient(&cfg.Cloud, printerMgr)
			s.pollClient.PrinterStatuses = s.getPrinterStatuses
			s.pollClient.PrinterList = s.getPrinterList
		}
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// Health check
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Registration
	s.mux.HandleFunc("POST /api/register", s.handleRegister)

	// Config
	s.mux.HandleFunc("GET /api/config", s.handleGetConfig)
	s.mux.HandleFunc("PUT /api/config", s.handleUpdateConfig)

	// Printer management
	s.mux.HandleFunc("GET /api/printers", s.handleListPrinters)
	s.mux.HandleFunc("POST /api/printers", s.handleAddPrinter)
	s.mux.HandleFunc("PUT /api/printers/{id}", s.handleUpdatePrinter)
	s.mux.HandleFunc("DELETE /api/printers/{id}", s.handleDeletePrinter)
	s.mux.HandleFunc("POST /api/printers/discover", s.handleDiscoverPrinters)
	s.mux.HandleFunc("POST /api/printers/{id}/test", s.handleTestPrint)

	// Print jobs
	s.mux.HandleFunc("POST /api/print", s.handlePrint)
	s.mux.HandleFunc("GET /api/jobs", s.handleGetJobs)

	// Logs
	s.mux.HandleFunc("GET /api/logs", s.handleGetLogs)

	// Status
	s.mux.HandleFunc("GET /api/status", s.handleStatus)

	// Web UI
	s.mux.HandleFunc("GET /", s.handleUI)
}

// Start starts the HTTP server and cloud connection
func (s *Server) Start() error {
	// Start cloud client
	s.startCloudClient()

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.logBuffer.LogInfo("HTTP server listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// startCloudClient starts the appropriate cloud client (WS or polling)
func (s *Server) startCloudClient() {
	hasCredentials := s.config.Cloud.ServerID != "" && s.config.Cloud.APIKey != ""

	if !hasCredentials {
		s.logBuffer.LogInfo("No cloud credentials configured. Register this server to enable cloud printing.")
		return
	}

	if s.wsClient != nil {
		s.logBuffer.LogInfo("Starting WebSocket connection to cloud...")
		s.wsClient.Start()
	} else if s.pollClient != nil {
		s.logBuffer.LogInfo("Starting HTTP polling (interval: %s)...", s.config.Cloud.PollInterval)
		s.pollClient.Start()
	}
}

// stopCloudClients stops all cloud clients
func (s *Server) stopCloudClients() {
	if s.wsClient != nil {
		s.wsClient.Stop()
		s.wsClient = nil
	}
	if s.pollClient != nil {
		s.pollClient.Stop()
		s.pollClient = nil
	}
}

// Stop gracefully stops the server
func (s *Server) Stop() {
	s.stopCloudClients()
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// --- Registration ---

type registerRequest struct {
	Tenant           string `json:"tenant"`
	RegistrationCode string `json:"registration_code"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "error": "INVALID_REQUEST", "message": "Invalid request body",
		})
		return
	}

	if req.Tenant == "" || req.RegistrationCode == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "error": "MISSING_FIELDS", "message": "Tenant and registration code are required",
		})
		return
	}

	s.logBuffer.LogInfo("Registering with cloud (tenant: %s)...", req.Tenant)

	// Call cloud registration endpoint
	cloudURL := s.config.Cloud.Endpoint + "/servers/register"
	body, _ := json.Marshal(map[string]string{"registration_code": req.RegistrationCode})

	httpReq, err := http.NewRequest("POST", cloudURL, bytes.NewReader(body))
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "error": "INTERNAL_ERROR", "message": "Failed to create request",
		})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-DB-Name", req.Tenant)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		s.logBuffer.LogError("Registration failed: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "error": "NETWORK_ERROR",
			"message": "Cannot reach JetSetGo servers. Check your internet connection and try again.",
		})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Try to parse cloud error
		var cloudErr map[string]interface{}
		errCode := "REGISTRATION_FAILED"
		errMsg := "Registration failed. Please try again."

		if json.Unmarshal(respBody, &cloudErr) == nil {
			if code, ok := cloudErr["error"].(string); ok {
				errCode = code
			}
			if msg, ok := cloudErr["message"].(string); ok {
				errMsg = msg
			}
			if detail, ok := cloudErr["detail"].(string); ok && errMsg == "" {
				errMsg = detail
			}
		}

		s.logBuffer.LogError("Registration failed: %s - %s", errCode, errMsg)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "error": errCode, "message": errMsg,
		})
		return
	}

	// Parse cloud success response
	var cloudResp struct {
		ServerID   string `json:"server_id"`
		APIKey     string `json:"api_key"`
		WSEndpoint string `json:"ws_endpoint"`
		Name       string `json:"name"`
		Location   string `json:"location"`
	}
	if err := json.Unmarshal(respBody, &cloudResp); err != nil {
		s.logBuffer.LogError("Failed to parse cloud response: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "error": "PARSE_ERROR", "message": "Invalid response from cloud",
		})
		return
	}

	// Update config
	s.configMu.Lock()
	s.config.Cloud.ServerID = cloudResp.ServerID
	s.config.Cloud.APIKey = cloudResp.APIKey
	s.config.Cloud.Tenant = req.Tenant
	if cloudResp.WSEndpoint != "" {
		s.config.Cloud.WSEndpoint = cloudResp.WSEndpoint
	}
	if cloudResp.Name != "" {
		s.config.Cloud.ServerName = cloudResp.Name
	}
	if cloudResp.Location != "" {
		s.config.Cloud.Location = cloudResp.Location
	}

	// Save config
	if s.config.ConfigPath != "" {
		if err := s.config.Save(s.config.ConfigPath); err != nil {
			s.logBuffer.LogError("Failed to save config: %v", err)
		}
	}
	s.configMu.Unlock()

	// Start cloud client now that we're registered
	s.stopCloudClients()
	if s.config.Cloud.UseWebSocket {
		s.wsClient = cloud.NewWSClient(&s.config.Cloud, s.printerManager)
		s.wsClient.Start()
		s.logBuffer.LogInfo("WebSocket client started after registration")
	} else {
		s.pollClient = cloud.NewPollClient(&s.config.Cloud, s.printerManager)
		s.pollClient.PrinterStatuses = s.getPrinterStatuses
			s.pollClient.PrinterList = s.getPrinterList
		s.pollClient.Start()
		s.logBuffer.LogInfo("Polling client started after registration")
	}

	s.logBuffer.LogInfo("Registration successful! Server ID: %s", cloudResp.ServerID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"server_id":   cloudResp.ServerID,
		"server_name": cloudResp.Name,
		"location":    cloudResp.Location,
	})
}

// --- Config ---

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	// Redact API key to prefix only
	apiKeyPrefix := ""
	if len(s.config.Cloud.APIKey) > 12 {
		apiKeyPrefix = s.config.Cloud.APIKey[:12] + "..."
	} else if s.config.Cloud.APIKey != "" {
		apiKeyPrefix = "***"
	}

	isRegistered := s.config.Cloud.ServerID != "" && s.config.Cloud.APIKey != ""

	printers := make([]map[string]interface{}, 0, len(s.config.Printers))
	for _, p := range s.config.Printers {
		pm := map[string]interface{}{
			"id": p.ID, "name": p.Name, "type": p.Type, "paper_width": p.PaperWidth,
		}
		if p.Type == "network" {
			pm["address"] = p.Address
			pm["port"] = p.Port
		}
		if p.Type == "usb" {
			pm["vendor_id"] = p.VendorID
			pm["product_id"] = p.ProductID
		}
		printers = append(printers, pm)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"server": map[string]interface{}{
			"port": s.config.Server.Port,
			"host": s.config.Server.Host,
		},
		"cloud": map[string]interface{}{
			"endpoint":             s.config.Cloud.Endpoint,
			"ws_endpoint":          s.config.Cloud.WSEndpoint,
			"server_id":            s.config.Cloud.ServerID,
			"tenant":               s.config.Cloud.Tenant,
			"server_name":          s.config.Cloud.ServerName,
			"location":             s.config.Cloud.Location,
			"api_key_prefix":       apiKeyPrefix,
			"use_websocket":        s.config.Cloud.UseWebSocket,
			"poll_interval":        s.config.Cloud.PollInterval.String(),
			"ws_reconnect_delay":   s.config.Cloud.WSReconnectDelay.String(),
			"ws_max_reconnect_delay": s.config.Cloud.WSMaxReconnect.String(),
			"ws_ping_interval":     s.config.Cloud.WSPingInterval.String(),
		},
		"printers":      printers,
		"config_path":   s.config.ConfigPath,
		"is_registered": isRegistered,
	})
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request body"})
		return
	}

	s.configMu.Lock()
	restartRequired := false

	if serverCfg, ok := body["server"].(map[string]interface{}); ok {
		if port, ok := serverCfg["port"].(float64); ok {
			if int(port) != s.config.Server.Port {
				restartRequired = true
			}
			s.config.Server.Port = int(port)
		}
		if host, ok := serverCfg["host"].(string); ok {
			if host != s.config.Server.Host {
				restartRequired = true
			}
			s.config.Server.Host = host
		}
	}

	wsChanged := false
	credentialsChanged := false

	if cloudCfg, ok := body["cloud"].(map[string]interface{}); ok {
		if v, ok := cloudCfg["use_websocket"].(bool); ok {
			if v != s.config.Cloud.UseWebSocket {
				wsChanged = true
			}
			s.config.Cloud.UseWebSocket = v
		}
		if v, ok := cloudCfg["server_name"].(string); ok {
			s.config.Cloud.ServerName = v
		}
		if v, ok := cloudCfg["location"].(string); ok {
			s.config.Cloud.Location = v
		}
		// Allow editing credentials directly
		if v, ok := cloudCfg["server_id"].(string); ok {
			if v != s.config.Cloud.ServerID {
				credentialsChanged = true
			}
			s.config.Cloud.ServerID = v
		}
		if v, ok := cloudCfg["api_key"].(string); ok && v != "" {
			credentialsChanged = true
			s.config.Cloud.APIKey = v
		}
		if v, ok := cloudCfg["tenant"].(string); ok {
			s.config.Cloud.Tenant = v
		}
		if v, ok := cloudCfg["ws_endpoint"].(string); ok && v != "" {
			s.config.Cloud.WSEndpoint = v
			wsChanged = true
		}
		if v, ok := cloudCfg["poll_interval"].(string); ok {
			if d, err := time.ParseDuration(v); err == nil {
				s.config.Cloud.PollInterval = d
			}
		}
		if v, ok := cloudCfg["ws_reconnect_delay"].(string); ok {
			if d, err := time.ParseDuration(v); err == nil {
				s.config.Cloud.WSReconnectDelay = d
			}
		}
		if v, ok := cloudCfg["ws_max_reconnect_delay"].(string); ok {
			if d, err := time.ParseDuration(v); err == nil {
				s.config.Cloud.WSMaxReconnect = d
			}
		}
		if v, ok := cloudCfg["ws_ping_interval"].(string); ok {
			if d, err := time.ParseDuration(v); err == nil {
				s.config.Cloud.WSPingInterval = d
			}
		}
	}

	// Handle re-register (clear credentials)
	if action, ok := body["action"].(string); ok && action == "re-register" {
		s.config.Cloud.ServerID = ""
		s.config.Cloud.APIKey = ""
		s.config.Cloud.Tenant = ""
		s.config.Cloud.ServerName = ""
		s.config.Cloud.Location = ""
		credentialsChanged = true
	}

	// Save config
	if s.config.ConfigPath != "" {
		if err := s.config.Save(s.config.ConfigPath); err != nil {
			s.configMu.Unlock()
			s.logBuffer.LogError("Failed to save config: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to save config"})
			return
		}
	}

	// Manage cloud client lifecycle at runtime
	needsRestart := wsChanged || credentialsChanged
	hasCredentials := s.config.Cloud.ServerID != "" && s.config.Cloud.APIKey != ""
	useWs := s.config.Cloud.UseWebSocket
	s.configMu.Unlock()

	if needsRestart {
		s.stopCloudClients()
		s.logBuffer.LogInfo("Cloud client stopped")

		if hasCredentials {
			if useWs {
				s.wsClient = cloud.NewWSClient(&s.config.Cloud, s.printerManager)
				s.wsClient.Start()
				s.logBuffer.LogInfo("WebSocket client started")
			} else {
				s.pollClient = cloud.NewPollClient(&s.config.Cloud, s.printerManager)
				s.pollClient.PrinterStatuses = s.getPrinterStatuses
			s.pollClient.PrinterList = s.getPrinterList
				s.pollClient.Start()
				s.logBuffer.LogInfo("Polling client started (interval: %s)", s.config.Cloud.PollInterval)
			}
		}
	}

	s.logBuffer.LogInfo("Configuration updated")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"restart_required": restartRequired,
	})
}

// --- Printers ---

func (s *Server) handleListPrinters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	printers := make([]map[string]interface{}, 0)
	for _, p := range s.config.Printers {
		status := "unknown"
		// Check status from printer manager
		if mgdPrinter, err := s.printerManager.GetPrinter(p.ID); err == nil {
			status = mgdPrinter.Status()
		}
		pm := map[string]interface{}{
			"id": p.ID, "name": p.Name, "type": p.Type,
			"status": status, "paper_width": p.PaperWidth,
		}
		if p.Type == "network" {
			pm["address"] = p.Address
			pm["port"] = p.Port
		}
		printers = append(printers, pm)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"printers": printers})
}

func (s *Server) handleAddPrinter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var p config.PrinterConfig
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request body"})
		return
	}

	if p.ID == "" || p.Name == "" || p.Type == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "id, name, and type are required"})
		return
	}

	if p.PaperWidth == 0 {
		p.PaperWidth = 80
	}

	s.configMu.Lock()
	// Check for duplicate ID
	for _, existing := range s.config.Printers {
		if existing.ID == p.ID {
			s.configMu.Unlock()
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Printer ID already exists"})
			return
		}
	}

	s.config.Printers = append(s.config.Printers, p)

	// Add to printer manager
	if p.Type == "network" {
		np := printer.NewNetworkPrinter(p.ID, p.Name, p.Address, p.Port)
		s.printerManager.AddPrinter(np)
	}

	// Save config
	if s.config.ConfigPath != "" {
		s.config.Save(s.config.ConfigPath)
	}
	s.configMu.Unlock()

	s.logBuffer.LogInfo("Printer added: %s (%s)", p.Name, p.ID)
	if s.pollClient != nil {
		s.pollClient.SyncPrinters()
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"printer": map[string]interface{}{
			"id": p.ID, "name": p.Name, "type": p.Type,
			"address": p.Address, "port": p.Port, "paper_width": p.PaperWidth,
		},
	})
}

func (s *Server) handleUpdatePrinter(w http.ResponseWriter, r *http.Request) {
	printerID := r.PathValue("id")
	w.Header().Set("Content-Type", "application/json")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request body"})
		return
	}

	s.configMu.Lock()
	found := false
	for i, p := range s.config.Printers {
		if p.ID == printerID {
			if v, ok := updates["name"].(string); ok {
				s.config.Printers[i].Name = v
			}
			if v, ok := updates["address"].(string); ok {
				s.config.Printers[i].Address = v
			}
			if v, ok := updates["port"].(float64); ok {
				s.config.Printers[i].Port = int(v)
			}
			if v, ok := updates["paper_width"].(float64); ok {
				s.config.Printers[i].PaperWidth = int(v)
			}

			// Recreate printer in manager if network settings changed
			if p.Type == "network" {
				s.printerManager.RemovePrinter(p.ID)
				np := printer.NewNetworkPrinter(
					s.config.Printers[i].ID,
					s.config.Printers[i].Name,
					s.config.Printers[i].Address,
					s.config.Printers[i].Port,
				)
				s.printerManager.AddPrinter(np)
			}

			found = true
			break
		}
	}

	if !found {
		s.configMu.Unlock()
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Printer not found"})
		return
	}

	if s.config.ConfigPath != "" {
		s.config.Save(s.config.ConfigPath)
	}
	s.configMu.Unlock()

	s.logBuffer.LogInfo("Printer updated: %s", printerID)
	if s.pollClient != nil {
		s.pollClient.SyncPrinters()
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Server) handleDeletePrinter(w http.ResponseWriter, r *http.Request) {
	printerID := r.PathValue("id")
	w.Header().Set("Content-Type", "application/json")

	s.configMu.Lock()
	found := false
	for i, p := range s.config.Printers {
		if p.ID == printerID {
			s.config.Printers = append(s.config.Printers[:i], s.config.Printers[i+1:]...)
			s.printerManager.RemovePrinter(printerID)
			found = true
			break
		}
	}

	if !found {
		s.configMu.Unlock()
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Printer not found"})
		return
	}

	if s.config.ConfigPath != "" {
		s.config.Save(s.config.ConfigPath)
	}
	s.configMu.Unlock()

	s.logBuffer.LogInfo("Printer removed: %s", printerID)
	if s.pollClient != nil {
		s.pollClient.SyncPrinters()
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleDiscoverPrinters scans for available printers
func (s *Server) handleDiscoverPrinters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.logBuffer.LogInfo("Scanning for network printers...")
	discovered, err := s.printerManager.Discover()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"discovered": []interface{}{}, "error": err.Error()})
		return
	}

	s.logBuffer.LogInfo("Found %d printer(s)", len(discovered))
	json.NewEncoder(w).Encode(map[string]interface{}{"discovered": discovered})
}

// handleTestPrint sends a test print to a printer
func (s *Server) handleTestPrint(w http.ResponseWriter, r *http.Request) {
	printerID := r.PathValue("id")
	w.Header().Set("Content-Type", "application/json")

	s.logBuffer.LogInfo("Sending test print to %s...", printerID)
	err := s.printerManager.TestPrint(printerID)
	if err != nil {
		s.logBuffer.LogError("Test print failed for %s: %v", printerID, err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	s.logBuffer.LogInfo("Test print successful: %s", printerID)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Test print sent successfully"})
}

// --- Print ---

// PrintRequest represents a print job request
type PrintRequest struct {
	PrinterID string `json:"printer_id"`
	Data      []byte `json:"data"`
}

func (s *Server) handlePrint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req PrintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Record job
	job := JobRecord{
		ID:        fmt.Sprintf("local_%d", time.Now().UnixMilli()),
		PrinterID: req.PrinterID,
		Status:    "printing",
		DataSize:  len(req.Data),
		CreatedAt: time.Now(),
	}

	// Find printer name
	s.configMu.RLock()
	for _, p := range s.config.Printers {
		if p.ID == req.PrinterID {
			job.PrinterName = p.Name
			break
		}
	}
	s.configMu.RUnlock()

	s.jobBuffer.Add(job)

	err := s.printerManager.Print(req.PrinterID, req.Data)
	if err != nil {
		s.jobBuffer.UpdateStatus(job.ID, "failed", err.Error())
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	s.jobBuffer.UpdateStatus(job.ID, "completed", "")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// --- Jobs ---

func (s *Server) handleGetJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"jobs": s.jobBuffer.Entries()})
}

// --- Logs ---

func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	levelFilter := r.URL.Query().Get("level")
	var levels []string
	if levelFilter != "" {
		levels = strings.Split(levelFilter, ",")
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"logs": s.logBuffer.Entries(levels)})
}

// --- Status ---

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.configMu.RLock()
	isRegistered := s.config.Cloud.ServerID != "" && s.config.Cloud.APIKey != ""
	tenant := s.config.Cloud.Tenant
	serverName := s.config.Cloud.ServerName
	printerCount := len(s.config.Printers)
	wsEnabled := s.config.Cloud.UseWebSocket
	s.configMu.RUnlock()

	cloudConnected := false
	connMethod := "none"
	wsStatus := map[string]interface{}{
		"enabled":      wsEnabled,
		"connected":    false,
		"reconnecting": false,
	}

	if s.wsClient != nil {
		connMethod = "websocket"
		status := s.wsClient.Status()
		cloudConnected = status.Connected
		wsStatus["connected"] = status.Connected
		wsStatus["reconnecting"] = status.Reconnecting
		if status.LastError != "" {
			wsStatus["last_error"] = status.LastError
		}
	} else if s.pollClient != nil {
		connMethod = "polling"
		status := s.pollClient.Status()
		cloudConnected = status.Connected
		wsStatus["connected"] = status.Connected
		if status.LastError != "" {
			wsStatus["last_error"] = status.LastError
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "running",
		"cloud_connected":   cloudConnected,
		"connection_method": connMethod,
		"websocket":         wsStatus,
		"printers_count":    printerCount,
		"is_registered":     isRegistered,
		"tenant":            tenant,
		"server_name":       serverName,
	})
}

// getPrinterList returns printer configs for cloud syncing
func (s *Server) getPrinterList() []map[string]interface{} {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	printers := make([]map[string]interface{}, 0, len(s.config.Printers))
	for _, p := range s.config.Printers {
		pw := p.PaperWidth
		if pw == 0 {
			pw = 80
		}
		printers = append(printers, map[string]interface{}{
			"printer_id":  p.ID,
			"name":        p.Name,
			"type":        p.Type,
			"paper_width": pw,
		})
	}
	return printers
}

// getPrinterStatuses returns current printer statuses for heartbeat reporting
func (s *Server) getPrinterStatuses() map[string]string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	statuses := make(map[string]string)
	for _, p := range s.config.Printers {
		status := "unknown"
		if mgdPrinter, err := s.printerManager.GetPrinter(p.ID); err == nil {
			status = mgdPrinter.Status()
		}
		statuses[p.ID] = status
	}
	return statuses
}

// --- Web UI ---

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(webUI))
}

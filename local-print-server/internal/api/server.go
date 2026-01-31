package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jetsetgo/local-print-server/internal/config"
	"github.com/jetsetgo/local-print-server/internal/printer"
)

// Server represents the HTTP server
type Server struct {
	config         *config.Config
	printerManager *printer.Manager
	mux            *http.ServeMux
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config) *Server {
	s := &Server{
		config:         cfg,
		printerManager: printer.NewManager(),
		mux:            http.NewServeMux(),
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

	s.setupRoutes()
	return s
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// Health check
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Printer management
	s.mux.HandleFunc("GET /api/printers", s.handleListPrinters)
	s.mux.HandleFunc("POST /api/printers/discover", s.handleDiscoverPrinters)
	s.mux.HandleFunc("POST /api/printers/{id}/test", s.handleTestPrint)

	// Print jobs
	s.mux.HandleFunc("POST /api/print", s.handlePrint)

	// Status
	s.mux.HandleFunc("GET /api/status", s.handleStatus)

	// Web UI
	s.mux.HandleFunc("GET /", s.handleUI)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	return http.ListenAndServe(addr, s.mux)
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// handleListPrinters returns configured printers and their status
func (s *Server) handleListPrinters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	printers := make([]map[string]interface{}, 0)
	for _, p := range s.config.Printers {
		printers = append(printers, map[string]interface{}{
			"id":     p.ID,
			"name":   p.Name,
			"type":   p.Type,
			"status": "unknown", // TODO: Check actual status
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"printers": printers,
	})
}

// handleDiscoverPrinters scans for available printers
func (s *Server) handleDiscoverPrinters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	discovered, err := s.printerManager.Discover()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"discovered": discovered,
	})
}

// handleTestPrint sends a test print to a printer
func (s *Server) handleTestPrint(w http.ResponseWriter, r *http.Request) {
	printerID := r.PathValue("id")

	w.Header().Set("Content-Type", "application/json")

	err := s.printerManager.TestPrint(printerID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Test print sent successfully",
	})
}

// PrintRequest represents a print job request
type PrintRequest struct {
	PrinterID string `json:"printer_id"`
	Data      []byte `json:"data"` // Base64-decoded ESC/POS bytes
}

// handlePrint handles incoming print jobs
func (s *Server) handlePrint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req PrintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := s.printerManager.Print(req.PrinterID, req.Data)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleStatus returns server status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "running",
		"cloud_connected": false, // TODO: Implement cloud connection check
		"printers_count":  len(s.config.Printers),
	})
}

// handleUI serves the web UI
func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(webUI))
}

// Embedded web UI (simple for now)
const webUI = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JetSetGo Print Server</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            color: #333;
            line-height: 1.6;
        }
        .container { max-width: 800px; margin: 0 auto; padding: 20px; }
        header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px 20px;
            text-align: center;
            margin-bottom: 30px;
            border-radius: 8px;
        }
        header h1 { font-size: 24px; margin-bottom: 5px; }
        header p { opacity: 0.9; font-size: 14px; }
        .card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .card h2 {
            font-size: 18px;
            margin-bottom: 15px;
            padding-bottom: 10px;
            border-bottom: 1px solid #eee;
        }
        .status-row {
            display: flex;
            justify-content: space-between;
            padding: 10px 0;
            border-bottom: 1px solid #f0f0f0;
        }
        .status-row:last-child { border-bottom: none; }
        .status-label { color: #666; }
        .status-value { font-weight: 500; }
        .status-online { color: #22c55e; }
        .status-offline { color: #ef4444; }
        .btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 14px;
            transition: background 0.2s;
        }
        .btn:hover { background: #5a67d8; }
        .btn-secondary {
            background: #e5e7eb;
            color: #374151;
        }
        .btn-secondary:hover { background: #d1d5db; }
        .printer-list { list-style: none; }
        .printer-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 15px;
            background: #f9fafb;
            border-radius: 6px;
            margin-bottom: 10px;
        }
        .printer-info h3 { font-size: 16px; margin-bottom: 3px; }
        .printer-info p { font-size: 13px; color: #666; }
        .no-printers {
            text-align: center;
            padding: 40px;
            color: #666;
        }
        #log {
            background: #1a1a2e;
            color: #a0aec0;
            padding: 15px;
            border-radius: 6px;
            font-family: monospace;
            font-size: 13px;
            max-height: 200px;
            overflow-y: auto;
        }
        .log-entry { margin-bottom: 5px; }
        .log-time { color: #667eea; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>JetSetGo Print Server</h1>
            <p>Local thermal printer gateway</p>
        </header>

        <div class="card">
            <h2>Server Status</h2>
            <div class="status-row">
                <span class="status-label">Server</span>
                <span class="status-value status-online" id="server-status">Running</span>
            </div>
            <div class="status-row">
                <span class="status-label">Cloud Connection</span>
                <span class="status-value status-offline" id="cloud-status">Not Connected</span>
            </div>
            <div class="status-row">
                <span class="status-label">Printers</span>
                <span class="status-value" id="printer-count">0 configured</span>
            </div>
        </div>

        <div class="card">
            <h2>Printers</h2>
            <ul class="printer-list" id="printer-list">
                <li class="no-printers">No printers configured. Click "Discover Printers" to scan.</li>
            </ul>
            <div style="margin-top: 15px; display: flex; gap: 10px;">
                <button class="btn" onclick="discoverPrinters()">Discover Printers</button>
                <button class="btn btn-secondary" onclick="refreshPrinters()">Refresh</button>
            </div>
        </div>

        <div class="card">
            <h2>Activity Log</h2>
            <div id="log">
                <div class="log-entry"><span class="log-time">[--:--:--]</span> Server started</div>
            </div>
        </div>
    </div>

    <script>
        function log(message) {
            const logEl = document.getElementById('log');
            const time = new Date().toLocaleTimeString();
            const entry = document.createElement('div');
            entry.className = 'log-entry';
            entry.innerHTML = '<span class="log-time">[' + time + ']</span> ' + message;
            logEl.appendChild(entry);
            logEl.scrollTop = logEl.scrollHeight;
        }

        async function fetchStatus() {
            try {
                const res = await fetch('/api/status');
                const data = await res.json();
                document.getElementById('printer-count').textContent = data.printers_count + ' configured';
                document.getElementById('cloud-status').textContent = data.cloud_connected ? 'Connected' : 'Not Connected';
                document.getElementById('cloud-status').className = 'status-value ' + (data.cloud_connected ? 'status-online' : 'status-offline');
            } catch (err) {
                log('Error fetching status: ' + err.message);
            }
        }

        async function refreshPrinters() {
            try {
                const res = await fetch('/api/printers');
                const data = await res.json();
                const list = document.getElementById('printer-list');

                if (data.printers.length === 0) {
                    list.innerHTML = '<li class="no-printers">No printers configured. Click "Discover Printers" to scan.</li>';
                } else {
                    list.innerHTML = data.printers.map(p =>
                        '<li class="printer-item">' +
                        '<div class="printer-info"><h3>' + p.name + '</h3><p>' + p.type + ' - ' + p.id + '</p></div>' +
                        '<button class="btn btn-secondary" onclick="testPrint(\'' + p.id + '\')">Test</button>' +
                        '</li>'
                    ).join('');
                }
                log('Printer list refreshed');
            } catch (err) {
                log('Error refreshing printers: ' + err.message);
            }
        }

        async function discoverPrinters() {
            log('Scanning for printers...');
            try {
                const res = await fetch('/api/printers/discover', { method: 'POST' });
                const data = await res.json();
                log('Found ' + data.discovered.length + ' printer(s)');
                refreshPrinters();
            } catch (err) {
                log('Error discovering printers: ' + err.message);
            }
        }

        async function testPrint(printerId) {
            log('Sending test print to ' + printerId + '...');
            try {
                const res = await fetch('/api/printers/' + printerId + '/test', { method: 'POST' });
                const data = await res.json();
                if (data.success) {
                    log('Test print successful!');
                } else {
                    log('Test print failed: ' + data.error);
                }
            } catch (err) {
                log('Error sending test print: ' + err.message);
            }
        }

        // Initial load
        fetchStatus();
        refreshPrinters();

        // Poll for status updates
        setInterval(fetchStatus, 5000);
    </script>
</body>
</html>`

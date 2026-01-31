# Print Server PRD

## Product Requirements Document
**Project:** JetSetGo Print Server System
**Version:** 1.0
**Date:** 2025-01-31
**Status:** Draft

---

## 1. Executive Summary

JetSetGo requires a printing solution that enables its SaaS platform to print receipts and tickets to thermal printers located at customer sites. This system consists of two components:

1. **Cloud Print Server** - Integrated into the existing FastAPI backend
2. **Local Print Server** - Lightweight Go application running on customer hardware

The cloud server handles all complex processing (ESC/POS command generation, formatting, image dithering). The local server is a thin relay that receives pre-generated print data and forwards it to physical printers.

---

## 2. Problem Statement

### Current State
- JetSetGo SaaS has no direct printing capability
- Customers need to print receipts, tickets, and boarding passes
- Web browsers cannot directly communicate with thermal printers
- Thermal printers use ESC/POS protocol, not standard print drivers

### Desired State
- Seamless printing from any JetSetGo web application
- Support for thermal receipt printers (58mm, 80mm)
- Works across customer network configurations
- Minimal customer IT involvement for setup

---

## 3. System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLOUD (JetSetGo Infrastructure)                │
│  ┌──────────────────┐      ┌──────────────────────────────────────────┐    │
│  │  JetSetGo SaaS   │─────►│  Cloud Print Server (FastAPI)            │    │
│  │  Web Apps        │      │  • Receives print requests (JSON)        │    │
│  └──────────────────┘      │  • Generates ESC/POS byte stream         │    │
│                            │  • Handles images, barcodes, QR codes    │    │
│                            │  • Pushes jobs via WebSocket             │    │
│                            │  • Manages printer registrations         │    │
│                            └─────────────────┬────────────────────────┘    │
└──────────────────────────────────────────────┼─────────────────────────────┘
                                               │
                              ┌────────────────┴────────────────┐
                              │  WebSocket (primary, ~50ms)     │
                              │  HTTP Polling (fallback, 30s)   │
                              └────────────────┬────────────────┘
                                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CUSTOMER SITE (Local Network)                     │
│  ┌──────────────────────────────────────────┐      ┌───────────────────┐   │
│  │  Local Print Server (Go)                 │      │  Thermal Printer  │   │
│  │  • WebSocket connection to cloud         │─────►│  (USB or Network) │   │
│  │  • Receives ESC/POS bytes instantly      │      └───────────────────┘   │
│  │  • Forwards to printer                   │                              │
│  │  • Reports status back to cloud          │      ┌───────────────────┐   │
│  │  • Falls back to polling if WS fails     │─────►│  Thermal Printer  │   │
│  └──────────────────────────────────────────┘      │  (Additional)     │   │
│                                                    └───────────────────┘   │
│  Hardware: Raspberry Pi Zero 2 W ($15) or any PC                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.1 Communication Strategy

| Method | Latency | Use Case |
|--------|---------|----------|
| **WebSocket (Primary)** | ~50ms | Real-time job delivery |
| **HTTP Polling (Fallback)** | 0-30s | When WebSocket unavailable |

**Why WebSocket?**
- Local server initiates connection (firewall-friendly)
- Cloud pushes jobs instantly over persistent connection
- No polling overhead when idle
- Automatic reconnection with exponential backoff

**Why keep polling fallback?**
- Some corporate firewalls block WebSocket
- Simpler debugging during development
- Graceful degradation

---

## 4. Component Specifications

### 4.1 Cloud Print Server (FastAPI)

**Responsibility:** All intelligent processing

| Feature | Description |
|---------|-------------|
| Print Job API | REST endpoint receiving structured print requests |
| ESC/POS Generation | Convert JSON receipt data to ESC/POS byte stream |
| Image Processing | Dither and convert images for thermal printing |
| Barcode/QR Generation | Generate barcode/QR code ESC/POS commands |
| Job Queue | Per-tenant, per-location job queuing |
| Printer Registry | Track registered local print servers |
| Job Status | Track pending, printing, completed, failed states |
| Retry Logic | Automatic retry for failed print jobs |

**API Endpoints (Cloud):**

```
POST /api/v1/print/jobs
  - Create a new print job (JSON receipt data)
  - Returns: job_id

GET /api/v1/print/jobs/{job_id}
  - Get job status
  - Returns: status, error details if failed

WS /api/v1/print/servers/{server_id}/ws
  - WebSocket endpoint for real-time job delivery (PRIMARY)
  - Auth: API key in connection header or query param
  - Server pushes: { type: "job", job_id, printer_id, data: "<base64 ESC/POS>" }
  - Client sends: { type: "status", job_id, status: "completed|failed", error? }
  - Client sends: { type: "ping" } to keep alive

GET /api/v1/print/servers/{server_id}/jobs
  - Get pending jobs for a local print server (FALLBACK polling endpoint)
  - Returns: array of jobs with ESC/POS bytes (base64)

POST /api/v1/print/servers/{server_id}/jobs/{job_id}/status
  - Local server reports job completion/failure (FALLBACK for non-WS)
  - Body: { status: "completed" | "failed", error?: string }

POST /api/v1/print/servers/register
  - Register a new local print server
  - Returns: server_id, api_key, ws_endpoint

GET /api/v1/print/servers/{server_id}/printers
  - List printers configured on local server
```

**WebSocket Message Types:**

```json
// Cloud → Local: New print job
{
  "type": "job",
  "job_id": "job_abc123",
  "printer_id": "receipt-1",
  "data": "<base64-encoded ESC/POS bytes>"
}

// Local → Cloud: Job status update
{
  "type": "status",
  "job_id": "job_abc123",
  "status": "completed",  // or "failed"
  "error": null           // error message if failed
}

// Local → Cloud: Heartbeat
{
  "type": "ping"
}

// Cloud → Local: Heartbeat response
{
  "type": "pong"
}
```

**Print Job Request Format (JSON):**

```json
{
  "printer_id": "kitchen-1",
  "template": "receipt",
  "data": {
    "header": {
      "logo": "base64_image_data",
      "business_name": "Island Ferry Co",
      "address": ["123 Harbor St", "Auckland, NZ"]
    },
    "items": [
      { "name": "Adult Return", "qty": 2, "price": 45.00 },
      { "name": "Child Return", "qty": 1, "price": 22.50 }
    ],
    "totals": {
      "subtotal": 112.50,
      "tax": 16.88,
      "total": 129.38
    },
    "footer": {
      "barcode": "TKT-20250131-001234",
      "qr_code": "https://jetsetgo.world/ticket/abc123",
      "message": "Thank you for travelling with us!"
    }
  },
  "options": {
    "paper_width": 80,
    "cut": true,
    "copies": 1
  }
}
```

### 4.2 Local Print Server (Go)

**Responsibility:** Thin relay only

| Feature | Description |
|---------|-------------|
| **WebSocket Connection** | Persistent connection to cloud for instant job delivery |
| Polling Fallback | Fall back to HTTP polling if WebSocket unavailable |
| Print Forwarding | Send ESC/POS bytes to configured printers |
| USB Support | Connect to USB thermal printers |
| Network Support | Connect to network printers (TCP port 9100) |
| Status Reporting | Report job success/failure to cloud |
| Web UI | Simple configuration interface |
| Auto-discovery | Detect available printers |
| Resilience | Queue jobs locally if cloud unreachable |
| **Auto-Reconnect** | Reconnect WebSocket with exponential backoff |

**Local Server Does NOT:**
- Generate ESC/POS commands
- Process images
- Parse receipt templates
- Handle complex business logic

**Configuration (config.yaml):**

```yaml
server:
  port: 8080

cloud:
  endpoint: "https://api.jetsetgo.world/api/v1/print"
  ws_endpoint: "wss://api.jetsetgo.world/api/v1/print/servers/{server_id}/ws"
  server_id: "srv_abc123"
  api_key: "key_xyz789"

  # WebSocket settings
  use_websocket: true           # Primary transport
  ws_reconnect_delay: 1s        # Initial reconnect delay
  ws_max_reconnect_delay: 30s   # Max reconnect delay (exponential backoff)
  ws_ping_interval: 30s         # Heartbeat interval

  # Polling fallback settings
  poll_interval: 30s            # Only used if WebSocket fails

printers:
  - id: "receipt-1"
    name: "Front Desk Receipt Printer"
    type: "usb"
    vendor_id: "0x04b8"  # Epson
    product_id: "0x0202"

  - id: "kitchen-1"
    name: "Kitchen Ticket Printer"
    type: "network"
    address: "192.168.1.100"
    port: 9100
```

**Local API Endpoints:**

```
GET /health
  - Health check

GET /api/printers
  - List configured printers and their status

POST /api/printers/discover
  - Scan for available printers

GET /api/status
  - Server status, cloud connection, job stats

GET /
  - Web UI for configuration
```

---

## 5. Hardware Requirements

### 5.1 Primary Target: Raspberry Pi Zero 2 W

| Specification | Value |
|---------------|-------|
| Price | $15 USD |
| CPU | Quad-core ARM Cortex-A53 @ 1GHz |
| RAM | 512 MB |
| Connectivity | WiFi 802.11 b/g/n, Bluetooth 4.2 |
| USB | 1x Micro-USB OTG (for printer) |
| Power | 5V via Micro-USB |
| Storage | microSD card |

**Additional Required:**
- microSD card (8GB+): ~$8
- Power supply (5V 2.5A): ~$8
- USB OTG adapter: ~$3
- **Total: ~$35 deployed**

### 5.2 Supported Platforms

| Platform | Architecture | Priority |
|----------|--------------|----------|
| Raspberry Pi Zero 2 W | linux/arm64 | Primary |
| Raspberry Pi Zero W | linux/arm/v6 | Secondary |
| Raspberry Pi 3/4/5 | linux/arm64 | Supported |
| Ubuntu/Debian x64 | linux/amd64 | Supported |
| Windows 10/11 | windows/amd64 | Supported |
| macOS (Intel) | darwin/amd64 | Development |
| macOS (Apple Silicon) | darwin/arm64 | Development |

---

## 6. Printer Compatibility

### 6.1 Target Printers

ESC/POS compatible thermal receipt printers:

| Brand | Models | Connection |
|-------|--------|------------|
| Epson | TM-T20, TM-T88 series | USB, Network |
| Star Micronics | TSP100, TSP650 | USB, Network |
| Bixolon | SRP-330, SRP-350 | USB, Network |
| Citizen | CT-S310 | USB, Network |
| Generic | 58mm/80mm ESC/POS | USB, Network |

### 6.2 Paper Widths

| Width | Characters (Font A) | Use Case |
|-------|---------------------|----------|
| 58mm | 32 chars | Small receipts |
| 80mm | 48 chars | Standard receipts, tickets |

---

## 7. Security

### 7.1 Authentication

- Local server authenticates to cloud using API key
- API key issued during server registration
- Keys can be revoked from cloud admin panel
- WebSocket connections authenticate via API key in header or query param

### 7.2 Communication

- All cloud communication over HTTPS/WSS (TLS encrypted)
- WebSocket uses WSS (WebSocket Secure) only
- Local server validates cloud SSL certificate
- No inbound connections required at customer site

### 7.3 Network Considerations

- Local server initiates all connections (firewall-friendly)
- WebSocket is outbound connection from customer site
- No port forwarding required at customer site
- Works behind NAT and most corporate firewalls
- Fallback to HTTPS polling if WebSocket blocked

---

## 8. Deployment

### 8.1 Local Server Distribution

| Format | Target |
|--------|--------|
| Single binary | All platforms |
| .deb package | Debian/Ubuntu/Raspberry Pi OS |
| .msi installer | Windows |
| Docker image | Advanced users |
| SD card image | Pi Zero (pre-configured) |

### 8.2 Installation Flow (Pi Zero)

1. Customer downloads SD card image
2. Writes image to microSD card
3. Edits wifi-config.txt with WiFi credentials
4. Boots Pi Zero
5. Opens web UI at http://printserver.local
6. Enters registration code from JetSetGo dashboard
7. Configures printers
8. Ready to print

### 8.3 Installation Flow (Windows/Linux)

1. Download binary/installer
2. Run installer or extract binary
3. Launch application
4. Opens web UI automatically
5. Enter registration code
6. Configure printers
7. Set to run on startup (optional)

---

## 9. User Interface

### 9.1 Local Server Web UI

Simple, mobile-friendly web interface:

**Pages:**
1. **Status** - Connection status, recent jobs, printer health
2. **Printers** - List printers, add/remove, test print
3. **Settings** - Cloud connection, server config
4. **Logs** - Recent activity for troubleshooting

**Design:**
- Single HTML file with embedded CSS/JS (no external dependencies)
- Works offline for basic configuration
- Responsive for phone/tablet access

### 9.2 Cloud Admin UI (JetSetGo Dashboard)

Integration into existing JetSetGo admin:

- List registered print servers
- View server status and last seen
- Generate registration codes
- View print job history
- Monitor error rates

---

## 10. Development Phases

### Phase 1: MVP (Local Server)

**Goal:** Working local print server

- [ ] Go project setup
- [ ] USB printer connection (Epson TM-T20 as reference)
- [ ] Network printer connection (TCP 9100)
- [ ] Simple HTTP API to receive print jobs
- [ ] Forward ESC/POS bytes to printer
- [ ] Basic web UI for configuration
- [ ] Cross-compile for Pi Zero
- [ ] Test on physical hardware

**Deliverable:** Local server that receives ESC/POS bytes via HTTP and prints them

### Phase 2: Cloud Integration

**Goal:** Full cloud-to-local pipeline

**Cloud (FastAPI):**
- [ ] Cloud print job API endpoints
- [ ] ESC/POS generation library (python-escpos or custom)
- [ ] Image dithering for thermal printers
- [ ] Barcode/QR code generation
- [ ] **WebSocket endpoint for real-time job push**
- [ ] HTTP polling endpoint (fallback)
- [ ] Job status tracking
- [ ] Server registration flow

**Local (Go):**
- [ ] **WebSocket client with auto-reconnect**
- [ ] Exponential backoff for reconnection
- [ ] Heartbeat/ping-pong handling
- [ ] HTTP polling fallback
- [ ] Job status reporting via WebSocket

**Deliverable:** End-to-end printing from JetSetGo app with <100ms latency

### Phase 3: Production Hardening

**Goal:** Production-ready system

- [ ] Comprehensive error handling
- [ ] Retry logic with backoff
- [ ] Offline job queuing
- [ ] Monitoring and alerting
- [ ] Customer documentation
- [ ] Installer packages
- [ ] SD card image for Pi Zero

**Deliverable:** Deployable product

---

## 11. Testing Strategy

### 11.1 Print Emulator

Use **Redbird ESC/POS Simulator** for development:
- Browser-based visual preview
- No physical printer needed
- Validates ESC/POS command output

Repository: https://github.com/Redbird-Corporation/ecspos-simulator

### 11.2 Physical Testing

Test matrix:

| Printer | Connection | Paper | Platform |
|---------|------------|-------|----------|
| Epson TM-T20 | USB | 80mm | Pi Zero, Windows |
| Epson TM-T88V | Network | 80mm | Pi Zero, Linux |
| Generic 58mm | USB | 58mm | Pi Zero |

---

## 12. Success Metrics

| Metric | Target |
|--------|--------|
| Print latency (cloud to paper) | **< 500ms** (WebSocket), < 3s (polling fallback) |
| WebSocket connection uptime | > 99% |
| Print success rate | > 99.5% |
| Local server uptime | > 99.9% |
| Setup time (customer) | < 15 minutes |
| Binary size | < 20 MB |
| Memory usage (Pi Zero) | < 50 MB |

---

## 13. Open Questions

1. ~~**WebSocket vs Polling**~~ - **RESOLVED**
   - **Decision:** WebSocket as primary transport, HTTP polling as fallback
   - **Rationale:** Sub-100ms latency with WebSocket, graceful fallback for restrictive networks

2. **Multiple Tenants** - Can one local server serve multiple JetSetGo tenants?
   - Recommendation: Yes, server registers with tenant, routes jobs by printer ID

3. **Offline Printing** - How long should local server queue jobs when cloud is unreachable?
   - Recommendation: 24 hours, then warn user

4. **Cash Drawer** - Should we support cash drawer kick commands?
   - Recommendation: Yes, simple addition to ESC/POS commands

---

## 14. References

- [ESC/POS Command Reference](https://reference.epson-biz.com/modules/ref_escpos/index.php)
- [Redbird ESC/POS Simulator](https://github.com/Redbird-Corporation/ecspos-simulator)
- [Go escpos library](https://github.com/hennedo/escpos)
- [Raspberry Pi Zero 2 W](https://www.raspberrypi.com/products/raspberry-pi-zero-2-w/)

---

## Appendix A: ESC/POS Command Examples

```
# Initialize printer
ESC @

# Bold on
ESC E 1

# Print text
"Hello World\n"

# Bold off
ESC E 0

# Center align
ESC a 1

# Print and feed
LF

# Cut paper
GS V 66 0
```

## Appendix B: Project Structure (Local Server)

```
local-print-server/
├── cmd/
│   └── printserver/
│       └── main.go           # Entry point
├── internal/
│   ├── api/
│   │   ├── handlers.go       # HTTP handlers
│   │   └── routes.go         # Route definitions
│   ├── cloud/
│   │   ├── client.go         # Cloud API client
│   │   ├── websocket.go      # WebSocket connection manager (PRIMARY)
│   │   └── polling.go        # Job polling logic (FALLBACK)
│   ├── printer/
│   │   ├── printer.go        # Printer interface
│   │   ├── usb.go            # USB printer implementation
│   │   ├── network.go        # Network printer implementation
│   │   └── discovery.go      # Printer auto-discovery
│   ├── config/
│   │   └── config.go         # Configuration management
│   └── ui/
│       └── embed.go          # Embedded web UI
├── web/
│   └── index.html            # Web UI (embedded)
├── configs/
│   └── config.example.yaml   # Example configuration
├── scripts/
│   ├── build-all.sh          # Cross-compilation script
│   └── package-deb.sh        # Debian package script
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
```

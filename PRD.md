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

A single-file embedded web interface served by the Go binary. No external CDN, no JS frameworks, no build step. Pure HTML + CSS + vanilla JavaScript. Mobile-friendly and responsive.

#### 9.1.1 Navigation & Layout

The UI uses a **single-page application** pattern with hash-based routing (`#/dashboard`, `#/printers`, `#/settings`, `#/logs`). Navigation is a horizontal tab bar pinned to the top, below the header.

**Header (persistent)**
- Purple gradient bar (`#667eea` to `#764ba2`) matching the existing Go UI
- "JetSetGo Print Server" title, left-aligned
- Small connection status dot in header, right-aligned (green = connected, red = disconnected, yellow = reconnecting)

**Tab Bar**
- Tabs: **Dashboard** | **Printers** | **Settings** | **Logs**
- Active tab has a blue bottom border and darker text
- Inactive tabs are gray
- On first run (no `server_id` in config), the tab bar is hidden and the Registration Wizard is shown full-screen

**Content Area**
- Below the tab bar, scrollable
- Max-width 900px, centered with padding on larger screens
- Cards use white backgrounds, 8px border-radius, subtle box-shadow

#### 9.1.2 First-Run Experience (Registration Wizard)

Shown automatically when the server has no `server_id` and `api_key` in its configuration. The wizard fills the entire content area with the tab bar hidden.

**Step Indicator**
- Three circles connected by lines at the top: `1 ── 2 ── 3`
- Active step: filled blue circle with white number
- Completed steps: green circle with checkmark
- Future steps: gray circle with gray number
- Matches the step indicator pattern from the admin UI's `AddServerPage.tsx`

**Step 1: Welcome**
- Large printer icon (SVG, no external dependencies)
- Heading: "Welcome to JetSetGo Print Server"
- Subtext: "Connect this server to your JetSetGo account to start printing receipts, tickets, and boarding passes."
- Single large blue button: "Get Started"
- Below the button, small gray text: "You'll need a registration code from your JetSetGo admin dashboard"

**Step 2: Connect to Cloud**
- Two input fields:
  - **Tenant ID** - Text input, placeholder "e.g., tta". Required. This maps to the `X-DB-Name` header value. Help text below: "Your organization's tenant identifier, provided by JetSetGo."
  - **Registration Code** - Large monospace input (6 characters), uppercase auto-transform, placeholder "ABC123". Required. Character-spaced for readability (letter-spacing: 0.5em, font-size: 2rem). Help text: "Enter the code shown in your JetSetGo admin dashboard."
- Blue button: "Register Server"
- "Back" text link below the button
- Error states displayed as a red alert box below the form:
  - Invalid code: "Registration code not found. Please check the code and try again."
  - Expired code: "Registration code has expired. Please generate a new one from your admin dashboard."
  - Network error: "Cannot reach JetSetGo servers. Check your internet connection and try again."
  - Already used: "This registration code has already been used."

**Step 3: Success**
- Large green circle with animated checkmark (CSS animation, draw-in effect)
- Heading: "Connected Successfully!"
- Subtext: "Your print server is now linked to JetSetGo. You can configure printers and start printing."
- Server name and tenant displayed in a small info card
- Blue button: "Go to Dashboard"
- Auto-redirects to dashboard after 3 seconds (with countdown text: "Redirecting in 3...")

**Registration API Call (Step 2 → Step 3)**
When the user clicks "Register Server":
1. Show a loading state on the button (spinner icon + "Connecting...")
2. Call `POST /api/register` on the local Go server, sending `{ tenant: "...", registration_code: "..." }`
3. The Go server forwards the code to the cloud endpoint `POST /api/v1/print/servers/register` with `X-DB-Name` header
4. On success: the cloud returns `server_id`, `api_key`, `ws_endpoint`; the Go server saves these to `config.yaml` and returns success
5. On failure: display the appropriate error message from the list above
6. On success: animate transition to Step 3

#### 9.1.3 Dashboard / Status Page

The main landing page after registration. Provides an at-a-glance overview of the print server's health.

**Connection Status Hero Card**
- Full-width card at the top
- Large status indicator: a colored circle (40px) with text beside it
  - **Connected** - Green circle with pulse animation. Text: "Connected to JetSetGo". Subtext: "WebSocket active" or "Polling every Xs"
  - **Disconnected** - Red circle. Text: "Disconnected". Subtext: "Last connected: [time ago]"
  - **Reconnecting** - Yellow circle with pulse animation. Text: "Reconnecting...". Subtext: "Attempt X, next retry in Ys"
- Right side of the card: server identity
  - Server name (from config)
  - Tenant ID
  - Server ID (truncated with tooltip for full value)

**Printer Summary Cards**
- Horizontal scrollable row on mobile, grid on desktop (2 or 3 columns)
- Each printer gets a card showing:
  - Printer name (bold)
  - Type badge: "Network" (blue) or "USB" (purple)
  - Paper width badge: "80mm" or "58mm" (gray outline)
  - Status dot: green (reachable) / red (unreachable) / gray (unknown)
  - Last job time (relative: "2 minutes ago" or "No jobs yet")
  - "Test Print" small button in the card footer

**Recent Jobs List**
- Section heading: "Recent Print Jobs"
- Displays last 20 jobs from the in-memory ring buffer
- Each row shows:
  - Job ID (truncated, monospace)
  - Printer name
  - Status badge (colored pill, matching admin UI colors):
    - `completed`: green pill
    - `failed`: red pill
    - `printing`: blue pill with pulse
    - `pending`: yellow pill
  - Timestamp (relative)
  - Data size (e.g., "2.1 KB")
- If no jobs: empty state with printer icon and "No print jobs yet. Send a test print to verify your setup."

**Quick Actions**
- Row of small buttons at the bottom of the dashboard:
  - "Test Print" (if at least 1 printer configured) - sends test print to the first configured printer
  - "Refresh Status" - re-fetches status from `/api/status`

**Auto-Refresh**
- The dashboard polls `/api/status` and `/api/jobs` every 5 seconds
- Status changes animate smoothly (CSS transitions on color, opacity)
- New jobs slide in from the top of the list

#### 9.1.4 Printers Page

Manages configured printers and allows discovery of new ones.

**Configured Printers List**
- Each printer in a card with:
  - Name (editable inline on click, or via edit button)
  - Type: "Network" or "USB"
  - Connection details: IP:Port for network, Vendor/Product ID for USB
  - Paper width indicator: "58mm" or "80mm" (selectable dropdown)
  - Status indicator: colored dot with text ("Online", "Offline", "Unknown")
  - Action buttons row:
    - "Test Print" (blue) - sends ESC/POS test page
    - "Edit" (gray) - opens edit form
    - "Remove" (red outline) - with confirmation dialog: "Remove [printer name]? This won't delete the physical printer."

**Add Printer Form**
- Expandable section at the top, toggled by "+ Add Printer" button
- Fields:
  - **Printer Name** - text input, required. Placeholder: "e.g., Front Desk Receipt"
  - **Type** - radio buttons: Network / USB
  - **IP Address** - shown when Network selected. Placeholder: "192.168.1.100"
  - **Port** - shown when Network selected. Default: 9100. Placeholder: "9100"
  - **Paper Width** - dropdown: 80mm (default) / 58mm
  - **Printer ID** - auto-generated from name (slugified), editable. Placeholder: "front-desk-receipt"
- "Add Printer" blue button, "Cancel" gray text button
- On success: printer appears in the list with a brief highlight animation

**Network Discovery**
- "Scan Network" button with a radar/scan icon
- When clicked:
  - Button shows spinner + "Scanning..."
  - Progress text: "Scanning 192.168.1.x..." (updates as subnets are scanned)
  - Results appear below as a list of found printers with IP:Port
  - Each result has an "Add" button that pre-fills the Add Printer form
- If no printers found: "No network printers found. Make sure printers are powered on and connected to the same network."

**Empty State**
- If no printers configured: large centered message
- Printer icon + "No printers configured"
- "Add a printer manually or scan your network to discover available printers."
- Two buttons: "+ Add Printer" and "Scan Network"

#### 9.1.5 Settings Page

Configuration management, organized in collapsible sections.

**Cloud Connection Section**
- Cloud endpoint URL (read-only display)
- WebSocket endpoint URL (read-only display)
- Connection method toggle: WebSocket (recommended) / HTTP Polling
  - When polling selected: poll interval input (number, seconds, min: 5, max: 300, default: 30)
- WebSocket reconnect settings (collapsed by default):
  - Initial reconnect delay (seconds)
  - Max reconnect delay (seconds)
  - Ping interval (seconds)

**Server Identity Section**
- Server Name - editable text input (current value pre-filled)
- Location - editable text input (optional, for operator reference)
- Server ID - read-only, displayed in monospace with copy button
- Tenant ID - read-only display
- API Key - shows prefix only (e.g., "ps_live_1SD..."), read-only
  - Note below: "To rotate the API key, use the JetSetGo admin dashboard."

**Local Server Section**
- HTTP Port - number input (default: 8080)
- Listen Address - text input (default: "0.0.0.0")
- Config file path - read-only display of where config.yaml is located

**Actions**
- "Save Changes" blue button (only enabled when changes are detected)
- "Reset to Defaults" gray button with confirmation dialog
- "Re-Register Server" red outline button (clears server_id and api_key, returns to registration wizard). Confirmation: "This will disconnect from JetSetGo and require a new registration code. Are you sure?"

**Save Behavior**
- Saves to config.yaml via `PUT /api/config`
- Success toast: "Settings saved. Some changes may require a server restart."
- If port or address changes: warning toast: "Server address changed. Restart the server for this to take effect."

#### 9.1.6 Logs Page

Real-time log viewer for troubleshooting.

**Log Viewer**
- Dark background container (`#1a1a2e`) with monospace font, matching the existing Activity Log style
- Each log entry on its own line:
  - Timestamp in blue (`#667eea`): `[2026-02-07 14:28:19]`
  - Log level in color: `INFO` (gray), `WARN` (yellow/orange), `ERROR` (red)
  - Message text in light gray (`#a0aec0`)
- Scrollable, max-height fills available viewport
- Most recent entries at the top (reverse chronological)

**Controls**
- **Filter buttons** (horizontal toggle group): All | Info | Warning | Error
  - Active filter has filled background, inactive are outlined
  - Filters apply immediately, animating out non-matching entries
- **Auto-scroll toggle**: checkbox + label "Auto-scroll to latest"
  - When enabled (default on), new log entries push the view to show the newest
  - When disabled, scroll position is preserved (useful when reading older entries)
- **Clear Logs** button (gray outline): "Clear all logs?" confirmation dialog
  - Only clears the UI display, not server-side ring buffer
- **Download Logs** button: downloads current visible logs as a `.txt` file

**Data Source**
- Polls `GET /api/logs` every 3 seconds
- The Go server maintains an in-memory ring buffer of the last 500 log entries
- Initial load fetches all buffered entries

#### 9.1.7 Design Constraints & Technical Requirements

**Single-File Embedding**
- The entire UI (HTML + CSS + JS) lives as a Go string constant in `internal/api/server.go`
- No external CDN, no font downloads, no image URLs
- All icons are inline SVG defined in the HTML
- Total size target: < 50 KB for the HTML string

**Styling**
- System font stack: `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`
- Monospace: `'SF Mono', 'Cascadia Code', 'Courier New', monospace`
- Color palette (matching existing Go UI and admin UI patterns):
  - Primary: `#667eea` (blue-purple)
  - Primary hover: `#5a67d8`
  - Success: `#22c55e`
  - Error: `#ef4444`
  - Warning: `#f59e0b`
  - Text primary: `#333` / `#1a1a2e`
  - Text secondary: `#666`
  - Background: `#f5f5f5`
  - Card background: `#ffffff`
  - Card border/shadow: `rgba(0,0,0,0.1)`
- Status badge colors match the admin UI `StatusBadge.tsx`:
  - Online/Ready/Completed: green bg `#dcfce7`, text `#166534`
  - Offline/Cancelled: gray bg `#f3f4f6`, text `#374151`
  - Pending/Busy: yellow bg `#fef9c3`, text `#854d0e`
  - Printing/Sent: blue bg `#dbeafe`, text `#1e40af`
  - Error/Failed: red bg `#fee2e2`, text `#991b1b`

**Animations (CSS only, no JS animation libraries)**
- Page transitions: `opacity` + `transform` with `transition: all 0.3s ease`
- Status dot pulse: CSS `@keyframes pulse` (scale 1 to 1.5, opacity 1 to 0)
- Card hover: subtle `box-shadow` increase, `transition: box-shadow 0.2s`
- Button hover: slight darken, `transition: background 0.2s`
- New list items: `@keyframes slideIn` (translateY -20px to 0, opacity 0 to 1)
- Success checkmark: `@keyframes drawCheck` using SVG stroke-dasharray/dashoffset
- Loading spinner: `@keyframes spin` (rotate 360deg, 0.8s linear infinite)

**Responsive Breakpoints**
- Mobile: < 640px - single column, full-width cards, stacked buttons
- Tablet: 640-900px - some cards side-by-side
- Desktop: > 900px - full grid layouts, max-width container

**Offline Capability**
- The UI is served from the Go binary, so it works even without internet
- Settings and printer configuration pages work fully offline
- Dashboard shows "Cloud: Disconnected" but still displays local printer status
- Logs page works fully (logs are local)

**Accessibility**
- All interactive elements are focusable with visible focus rings
- Buttons have `aria-label` where icon-only
- Form fields have associated `<label>` elements
- Status colors are supplemented with text labels (never color-only)
- Tab navigation works logically through the interface
- Confirmation dialogs trap focus appropriately

### 9.2 Local Server Go API Endpoints (for UI)

In addition to the existing endpoints (`/health`, `/api/printers`, `/api/printers/discover`, `/api/printers/{id}/test`, `/api/print`, `/api/status`), the following endpoints are required for the full web UI:

```
POST /api/register
  - Body: { "tenant": "tta", "registration_code": "ABC123" }
  - Calls cloud API: POST {cloud_endpoint}/servers/register with X-DB-Name header
  - On success: saves server_id, api_key, tenant, ws_endpoint to config.yaml
  - Returns: { "success": true, "server_id": "...", "server_name": "..." }
  - Errors: { "success": false, "error": "INVALID_CODE", "message": "..." }

GET /api/config
  - Returns current configuration (API key redacted to prefix only)
  - Response: {
      "server": { "port": 8080, "host": "0.0.0.0" },
      "cloud": {
        "endpoint": "...", "ws_endpoint": "...", "server_id": "...",
        "tenant": "...", "api_key_prefix": "ps_live_1SD...",
        "use_websocket": true, "poll_interval": "30s",
        "ws_reconnect_delay": "1s", "ws_max_reconnect_delay": "30s",
        "ws_ping_interval": "30s"
      },
      "printers": [...],
      "config_path": "/path/to/config.yaml",
      "is_registered": true
    }

PUT /api/config
  - Body: partial config object (only fields being changed)
  - Saves to config.yaml
  - Returns: { "success": true, "restart_required": false }
  - If port or host changed: restart_required = true

POST /api/printers
  - Body: { "id": "receipt-1", "name": "Front Desk", "type": "network",
            "address": "192.168.1.100", "port": 9100, "paper_width": 80 }
  - Adds printer to config and in-memory printer manager
  - Returns: { "success": true, "printer": { ... } }

PUT /api/printers/{id}
  - Body: partial printer config
  - Updates printer config and reloads in printer manager
  - Returns: { "success": true, "printer": { ... } }

DELETE /api/printers/{id}
  - Removes printer from config and printer manager
  - Returns: { "success": true }

GET /api/jobs
  - Returns recent jobs from in-memory ring buffer (last 50)
  - Response: { "jobs": [
      { "id": "job_abc123", "printer_id": "receipt-1", "printer_name": "Front Desk",
        "status": "completed", "data_size": 2048, "created_at": "...",
        "completed_at": "...", "error": null }
    ]}

GET /api/logs
  - Query param: ?level=info,warn,error (optional filter)
  - Returns recent log entries from in-memory ring buffer (last 500)
  - Response: { "logs": [
      { "timestamp": "2026-02-07T14:28:19Z", "level": "info",
        "message": "Print job completed: job_abc123" }
    ]}
```

### 9.3 Cloud Admin UI (JetSetGo Dashboard)

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

# Developer Prompt: Implement Local Print Server Web UI

## Context

You are working on the **JetSetGo Local Print Server** - a Go application that runs on customer hardware (Raspberry Pi, Windows PC, etc.) and relays print jobs from the JetSetGo cloud platform to thermal printers. The server currently has a basic embedded HTML status page. Your job is to build the full web UI as specified in the PRD.

## Requirements Document

Read the full PRD for context and detailed UX specifications:
- **`C:\Users\mail\Desktop\jetsetgo\print server\PRD.md`** - Sections 9.1 through 9.2 contain the comprehensive UX specs you need to implement

Pay special attention to:
- **Section 9.1.1** - Navigation & Layout (hash-based SPA routing)
- **Section 9.1.2** - First-Run Registration Wizard (3-step flow)
- **Section 9.1.3** - Dashboard / Status Page
- **Section 9.1.4** - Printers Page
- **Section 9.1.5** - Settings Page
- **Section 9.1.6** - Logs Page
- **Section 9.1.7** - Design Constraints (colors, animations, responsiveness, accessibility)
- **Section 9.2** - Go API Endpoints Required for UI

## Existing Codebase

The Go project is at: `C:\Users\mail\Desktop\jetsetgo\print server\local-print-server\`

### Key files to read first:
1. **`internal/api/server.go`** - HTTP server, route setup, existing handlers, and the current embedded HTML UI (the `webUI` const at line 222). This is where the new HTML will go.
2. **`internal/config/config.go`** - Config struct (`Config`, `CloudConfig`, `PrinterConfig`), `Load()` and `Save()` methods. You need to add a `Tenant` field to `CloudConfig` and a `PaperWidth` field to `PrinterConfig`.
3. **`internal/printer/manager.go`** - Printer management, discovery, test prints. Already has `AddPrinter()`, `TestPrint()`, `Discover()` methods.
4. **`internal/printer/network.go`** - Network printer driver (TCP port 9100 connections).
5. **`internal/cloud/websocket.go`** - WebSocket client with auto-reconnect. Has a `Status()` method that returns connection state.
6. **`cmd/printserver/main.go`** - Entry point, loads config, starts server.
7. **`config.yaml`** - Current active configuration (has server_id and api_key already set for testing).

### Admin UI files (for visual reference only - do NOT modify these):
These React components show the patterns and visual language your vanilla HTML/CSS should match:
- **`C:\Users\mail\Desktop\jetsetgo\app\jsg_printers\src\components\print\RegistrationCodeDisplay.tsx`** - How registration codes look in the admin UI (large monospace characters, glow effect, countdown timer)
- **`C:\Users\mail\Desktop\jetsetgo\app\jsg_printers\src\components\print\common\StatusBadge.tsx`** - Status badge color scheme you should replicate:
  - Online/Ready/Completed: green (`bg-green-100, text-green-800`)
  - Offline/Cancelled: gray (`bg-gray-100, text-gray-700`)
  - Pending/Busy: yellow (`bg-yellow-100, text-yellow-800`)
  - Printing/Sent: blue (`bg-blue-100, text-blue-800`)
  - Error/Failed: red (`bg-red-100, text-red-800`)
- **`C:\Users\mail\Desktop\jetsetgo\app\jsg_printers\src\components\print\pages\AddServerPage.tsx`** - The admin-side registration wizard flow (3 steps: details, code, success). Your local server's registration wizard is the "other side" of this flow.

## Critical Constraint: Single Embedded HTML File

The entire web UI must be a **single HTML string** embedded as a Go const in `server.go`. This means:
- **No external dependencies** - no CDN links, no external fonts, no external scripts
- **No build step** - no webpack, no npm, no TypeScript
- **Pure vanilla** - HTML + `<style>` + `<script>` in one file
- **All icons are inline SVG** - define them as JS functions or template literals
- **Target size: < 50 KB** for the entire HTML string

All Go string escaping rules apply (backtick strings can't contain backticks, so if you need backticks in the JS/HTML, use `\x60` or restructure to avoid them).

## Implementation Order

Build and test in this sequence:

### Phase 1: Go API Endpoints (build all the backend endpoints first)

1. **Add `Tenant` field to `CloudConfig`** in `config.go` and `PaperWidth` field to `PrinterConfig`
2. **Implement `POST /api/register`** - The registration handler that:
   - Accepts `{ tenant, registration_code }` from the UI
   - Calls the cloud endpoint `POST https://api.jetsetgo.world/api/v1/print/servers/register` with `X-DB-Name: <tenant>` header and `{ registration_code }` body
   - On success: receives `{ server_id, api_key, ws_endpoint, name, location }` from the cloud
   - Saves `server_id`, `api_key`, `tenant`, `ws_endpoint` to config.yaml
   - Returns success with server details
   - On failure: returns the cloud's error message
3. **Implement `GET /api/config`** - Returns current config with api_key redacted to prefix only
4. **Implement `PUT /api/config`** - Updates config fields and saves to config.yaml
5. **Implement `POST /api/printers`** - Adds a new printer to config + printer manager
6. **Implement `PUT /api/printers/{id}`** - Updates printer config
7. **Implement `DELETE /api/printers/{id}`** - Removes printer
8. **Implement `GET /api/jobs`** - Add an in-memory ring buffer (capacity 50) to the Server struct, populated when jobs complete via WebSocket or polling. Return recent jobs.
9. **Implement `GET /api/logs`** - Add an in-memory ring buffer (capacity 500) for log entries. Create a `LogBuffer` that captures Go `log` output. Return entries with level filtering via query param.
10. **Update `GET /api/status`** to include `is_registered` boolean and `tenant` field

### Phase 2: Registration Wizard UI (first thing the user sees)

Build the registration wizard first because:
- It's the first-run experience - needs to work perfectly
- It tests the `/api/register` and `/api/config` endpoints
- You can verify the full cloud registration flow end-to-end

Implement:
- SPA router (hash-based: `#/wizard`, `#/dashboard`, etc.)
- Step indicator component
- Step 1: Welcome screen
- Step 2: Registration form with tenant + code inputs
- Step 3: Success screen with animated checkmark
- Error handling for all registration failure cases
- Auto-detection of registration state (`GET /api/config` → `is_registered`)

### Phase 3: Dashboard Page

- Connection status hero card
- Printer summary cards
- Recent jobs list with status badges
- Quick actions (test print, refresh)
- Auto-refresh polling (5-second interval)

### Phase 4: Printers Page

- Configured printers list
- Add printer form
- Edit/remove printer
- Network discovery with scan button
- Test print per printer

### Phase 5: Settings Page

- Cloud connection settings
- Server identity (name, location)
- Local server settings (port, host)
- Save/reset/re-register actions

### Phase 6: Logs Page

- Log viewer with dark theme
- Level filtering
- Auto-scroll toggle
- Download logs as .txt

## Testing the Registration Flow

To test registration end-to-end:

1. Clear the `server_id` and `api_key` from `config.yaml` to trigger first-run mode
2. Start the Go server: `cd local-print-server && go run ./cmd/printserver/`
3. Open `http://localhost:8080` - should show the registration wizard
4. In a separate terminal, generate a registration code via the cloud API:
   ```bash
   # First get a Firebase token
   curl -X POST "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=AIzaSyCCk8B1djMgX8FTmoQt8fCfn3GDimeIxB4" -H "Content-Type: application/json" -d '{"email":"manus@secretagentsocks.com","password":"claude123","returnSecureToken":true}'

   # Then generate a registration code (use the idToken from above)
   curl -X POST "https://api.jetsetgo.world/api/v1/print/servers/registration-codes" \
     -H "Authorization: Bearer <idToken>" \
     -H "X-DB-Name: tta" \
     -H "Content-Type: application/json" \
     -d '{"name": "Test Server", "location": "Dev Machine"}'
   ```
5. Enter tenant "tta" and the registration code in the wizard
6. Verify the server registers successfully and transitions to the dashboard

## Code Style Notes

- The Go code follows standard Go conventions with unexported helpers and exported API types
- Error responses use `json.NewEncoder(w).Encode(map[string]interface{}{...})` pattern
- Route registration uses Go 1.22+ method-based patterns: `s.mux.HandleFunc("GET /path", handler)`
- Configuration is managed through the `config.Config` struct with YAML serialization
- The printer manager uses a mutex-protected map of printer interfaces

## What NOT to Do

- Do NOT add any external dependencies (npm packages, CDN links, Go modules for UI)
- Do NOT split the HTML into separate files - it must remain as a single embedded const
- Do NOT use Go templates (html/template) - the HTML is a static string with JS doing all dynamic rendering
- Do NOT modify the admin UI React components (they're just for visual reference)
- Do NOT change the cloud API endpoints - they're already production
- Do NOT add authentication to the local server's web UI (it's on a local network only)

# WebSocket 403 Fix - Server Admin Instructions

## Problem
The Go print server connects via WebSocket to `wss://api.jetsetgo.world/api/v1/print/servers/{server_id}/ws` but receives HTTP 403 Forbidden. The API key is confirmed valid.

## Root Cause (Most Likely)
The reverse proxy is routing WebSocket requests to the **API container (port 8001)** instead of the **background worker container (port 8002)**.

The API container has `ENABLE_BACKGROUND_WORKERS=false`, so the WebSocket handler immediately rejects with 403 before even checking auth. This explains why:
- No auth-related logs appear (the handler exits before reaching auth logic)
- The response is always 403 regardless of valid credentials
- The response body is "Invalid response status" (generic WebSocket rejection)

## What Needs to Happen

### 1. Deploy Code Changes (adds diagnostic logging)
Changes have been made to:
- `main.py` - Added detailed logging at every WebSocket rejection point
- `requirements.txt` - Added explicit `websockets>=12.0` (redundant but explicit)

**This has already been deployed** via the deployment webhook. Both containers were restarted with the new code.

### 2. Fix Reverse Proxy Routing (CRITICAL)
The WebSocket path **MUST** route to the background worker (port 8002), not the API container (port 8001).

**Path to route:** `/api/v1/print/servers/*/ws`

Example Nginx config:
```nginx
# WebSocket connections for print servers -> background worker
location ~ ^/api/v1/print/servers/.+/ws$ {
    proxy_pass http://jetsetgo-background-worker:8002;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
}
```

Example Caddy config:
```
@print_ws path_regexp ^/api/v1/print/servers/.+/ws$
handle @print_ws {
    reverse_proxy jetsetgo-background-worker:8002
}
```

**Key requirements:**
- `proxy_http_version 1.1` (required for WebSocket upgrade)
- `Upgrade` and `Connection` headers must be passed through
- Timeout should be at least 120s (print servers send pings every 30s)
- Route specifically to port **8002** (background worker), NOT port **8001** (API)

### 3. If Using Cloudflare
Ensure WebSocket is enabled for the domain:
- Cloudflare Dashboard > Domain > Network > WebSockets: **ON**
- Cloudflare should proxy WebSocket connections transparently when it sees the `Upgrade: websocket` header

### 4. Verify After Deployment
After deploying and fixing the proxy, check the logs:

```bash
# Check background worker logs for WS connection attempts (use logs API token from CLAUDE_SECRETS.env)
curl -H "Authorization: Bearer <LOGS_API_TOKEN>" \
  "https://logs.jetsetgo.world/api/v1/fastapi/logs/recent?seconds=120&limit=20"
```

**Expected log messages after fix:**
- `[WS] Incoming WebSocket connection for server_id=...` (request reached background worker)
- `[WS] Auth headers: tenant=..., api_key_present=True, ...` (headers received)
- `[WS] Auth PASSED for tenant=... server_id=...` (auth validated)
- `[WS] Connection ACCEPTED for tenant=... server_id=...` (fully connected)

**If you still see 403 but NO `[WS]` logs in background worker**, the request is still being routed to the API container.

**If you see `[WS] REJECTED: ENABLE_BACKGROUND_WORKERS=false`**, the request is explicitly hitting the API container. Fix the reverse proxy.

**If you see `[WS] REJECTED: Invalid API key`**, there's a database connectivity issue from the background worker. Check PgBouncer access from that container.

## Architecture Reference

| Container | Port | Purpose |
|-----------|------|---------|
| `jetsetgo-ticketing-api` | 8001 | HTTP API requests (Gunicorn, 14 workers) |
| `jetsetgo-background-worker` | 8002 | WebSocket + background tasks (Uvicorn, 1 worker) |

Both containers run the same codebase. The `ENABLE_BACKGROUND_WORKERS` env var controls behavior:
- API container: `ENABLE_BACKGROUND_WORKERS=false` -> rejects all WebSocket connections
- Background worker: `ENABLE_BACKGROUND_WORKERS=true` -> accepts WebSocket connections

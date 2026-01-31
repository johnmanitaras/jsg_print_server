# JetSetGo Print Server - Setup Guide

This document covers the setup, configuration, and usage of the JetSetGo local print server system.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Development Environment Setup](#development-environment-setup)
4. [Project Structure](#project-structure)
5. [Configuration](#configuration)
6. [Running the Server](#running-the-server)
7. [API Reference](#api-reference)
8. [Testing with ESC/POS Emulator](#testing-with-escpos-emulator)
9. [Cross-Compilation](#cross-compilation)
10. [Deployment](#deployment)

---

## Overview

The JetSetGo Print Server system enables thermal receipt printing from the JetSetGo SaaS platform. It consists of two components:

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Cloud Print Server** | FastAPI (Python) | Generates ESC/POS commands, manages job queue |
| **Local Print Server** | Go | Receives print jobs, forwards to physical printers |

This guide focuses on the **Local Print Server**.

### Key Features

- Single binary with no external dependencies
- Network printer support (TCP port 9100)
- USB printer support (coming soon)
- Web-based configuration UI
- Cross-platform (Windows, Linux, macOS, Raspberry Pi)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CLOUD (JetSetGo)                                │
│  ┌──────────────────┐      ┌─────────────────────────────────────┐     │
│  │  JetSetGo SaaS   │─────►│  Cloud Print Server (FastAPI)       │     │
│  │  Web Apps        │      │  • Generates ESC/POS byte stream    │     │
│  └──────────────────┘      │  • Manages print job queue          │     │
│                            └──────────────────┬──────────────────┘     │
└───────────────────────────────────────────────┼─────────────────────────┘
                                                │ HTTPS (polling)
                                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      CUSTOMER SITE (Local Network)                      │
│  ┌─────────────────────────────────────┐      ┌───────────────────┐    │
│  │  Local Print Server (Go)            │      │  Thermal Printer  │    │
│  │  • Polls cloud for jobs             │─────►│  Port 9100        │    │
│  │  • Forwards ESC/POS to printer      │      └───────────────────┘    │
│  │  • Reports status back              │                               │
│  │  • Web UI at :8080                  │                               │
│  └─────────────────────────────────────┘                               │
│                                                                         │
│  Hardware: Raspberry Pi Zero 2 W ($15) or any PC                       │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Development Environment Setup

### Prerequisites

- Windows 10/11, macOS, or Linux
- Git
- Docker Desktop (for ESC/POS emulator)

### Installing Go

Go 1.23.5 is installed at:

| Item | Path |
|------|------|
| GOROOT | `C:\Users\mail\go` |
| GOPATH | `C:\Users\mail\go-workspace` |
| Binary | `C:\Users\mail\go\bin\go.exe` |

**Verify installation:**

```bash
go version
# Output: go version go1.23.5 windows/amd64
```

**Environment variables** (already configured in `.bashrc` and Windows User PATH):

```bash
export GOROOT="/c/Users/mail/go"
export GOPATH="/c/Users/mail/go-workspace"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"
```

---

## Project Structure

```
print server/
├── PRD.md                              # Product Requirements Document
├── SETUP_GUIDE.md                      # This file
├── local-print-server/                 # Go application
│   ├── cmd/
│   │   └── printserver/
│   │       └── main.go                 # Entry point
│   ├── internal/
│   │   ├── api/
│   │   │   └── server.go               # HTTP server & web UI
│   │   ├── config/
│   │   │   └── config.go               # Configuration management
│   │   └── printer/
│   │       ├── manager.go              # Printer management
│   │       └── network.go              # Network printer driver
│   ├── configs/
│   │   └── config.example.yaml         # Example configuration
│   ├── bin/                            # Compiled binaries
│   │   ├── printserver.exe             # Windows
│   │   └── printserver-linux-arm64     # Raspberry Pi
│   ├── config.yaml                     # Active configuration
│   ├── go.mod
│   ├── go.sum
│   ├── Makefile
│   └── README.md
├── escpos-netprinter/                  # ESC/POS emulator (cloned)
└── escpos-emulator/                    # Redbird simulator (cloned)
```

---

## Configuration

### Configuration File

Create `config.yaml` in the `local-print-server` directory:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

cloud:
  endpoint: "https://api.jetsetgo.world/api/v1/print"
  server_id: ""        # Assigned during registration
  api_key: ""          # Assigned during registration
  poll_interval: 2s

printers:
  # Network printer example
  - id: "receipt-1"
    name: "Front Desk Printer"
    type: "network"
    address: "192.168.1.100"
    port: 9100

  # ESC/POS emulator (for testing)
  - id: "emulator-1"
    name: "ESC/POS Emulator"
    type: "network"
    address: "127.0.0.1"
    port: 9100
```

### Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `server.port` | int | 8080 | HTTP server port |
| `server.host` | string | "0.0.0.0" | Bind address |
| `cloud.endpoint` | string | - | Cloud print server URL |
| `cloud.server_id` | string | - | Registered server ID |
| `cloud.api_key` | string | - | API authentication key |
| `cloud.poll_interval` | duration | 2s | Cloud polling interval |
| `printers[].id` | string | - | Unique printer identifier |
| `printers[].name` | string | - | Human-readable name |
| `printers[].type` | string | - | "network" or "usb" |
| `printers[].address` | string | - | IP address (network only) |
| `printers[].port` | int | 9100 | TCP port (network only) |

---

## Running the Server

### Build

```bash
cd "print server/local-print-server"

# Build for current platform
go build -o bin/printserver.exe ./cmd/printserver

# Or use make
make build
```

### Run

```bash
# Run the server
./bin/printserver.exe

# Output:
# JetSetGo Local Print Server
# ===========================
# Server Port: 8080
# Cloud Endpoint: https://api.jetsetgo.world/api/v1/print
#
# Starting server on http://localhost:8080
# Press Ctrl+C to stop
```

### Access Web UI

Open http://localhost:8080 in your browser.

The web UI provides:
- Server status overview
- Printer list and status
- Test print functionality
- Activity log

---

## API Reference

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy"
}
```

### Server Status

```http
GET /api/status
```

**Response:**
```json
{
  "status": "running",
  "cloud_connected": false,
  "printers_count": 1
}
```

### List Printers

```http
GET /api/printers
```

**Response:**
```json
{
  "printers": [
    {
      "id": "emulator-1",
      "name": "ESC/POS Emulator",
      "type": "network",
      "status": "unknown"
    }
  ]
}
```

### Discover Printers

Scans local network for printers on port 9100.

```http
POST /api/printers/discover
```

**Response:**
```json
{
  "discovered": [
    {
      "id": "network-192.168.1.100",
      "name": "Printer at 192.168.1.100",
      "type": "network",
      "address": "192.168.1.100",
      "port": 9100
    }
  ]
}
```

### Test Print

Sends a test receipt to a printer.

```http
POST /api/printers/{id}/test
```

**Response:**
```json
{
  "success": true,
  "message": "Test print sent successfully"
}
```

### Print Job

Send raw ESC/POS data to a printer.

```http
POST /api/print
Content-Type: application/json

{
  "printer_id": "receipt-1",
  "data": "<base64-encoded ESC/POS bytes>"
}
```

**Response:**
```json
{
  "success": true
}
```

---

## Testing with ESC/POS Emulator

### Start the Emulator

The ESC/POS emulator runs as a Docker container:

```bash
# Start emulator (first time)
docker run -d --name escpos-emulator \
    -p 9100:9100/tcp \
    -p 8888:80/tcp \
    --env ESCPOS_DEBUG=True \
    gilbertfl/escpos-netprinter:3.2

# Start existing container
docker start escpos-emulator

# Stop emulator
docker stop escpos-emulator

# View logs
docker logs escpos-emulator
```

### Emulator Ports

| Port | Purpose |
|------|---------|
| 9100 | ESC/POS printer port (JetDirect) |
| 8888 | Web UI to view printed receipts |

### View Printed Receipts

1. Open http://localhost:8888 in browser
2. Click "Open printed receipt list"
3. Click on a receipt to view rendered output

### Test Workflow

```bash
# 1. Start emulator
docker start escpos-emulator

# 2. Start print server (with emulator in config)
cd "print server/local-print-server"
./bin/printserver.exe

# 3. Send test print
curl -X POST http://localhost:8080/api/printers/emulator-1/test

# 4. View receipt at http://localhost:8888/recus
```

### Sample Test Receipt Output

The test print generates:

```
┌─────────────────────────────┐
│         JETSETGO            │  ← Double size, bold
│        Print Server         │
│     -------------------     │
│                             │
│  Test Print                 │
│  Time: 2026-01-31 14:28:19  │
│                             │
│     -------------------     │
│        Printer OK!          │
│                             │
│         ═══════             │  ← Paper cut
└─────────────────────────────┘
```

---

## Cross-Compilation

Build binaries for all target platforms:

```bash
cd "print server/local-print-server"

# Raspberry Pi Zero 2 W / Pi 3/4/5 (ARM64)
GOOS=linux GOARCH=arm64 go build -o bin/printserver-linux-arm64 ./cmd/printserver

# Original Raspberry Pi Zero W (ARM v6)
GOOS=linux GOARCH=arm GOARM=6 go build -o bin/printserver-linux-arm ./cmd/printserver

# Linux x64
GOOS=linux GOARCH=amd64 go build -o bin/printserver-linux-amd64 ./cmd/printserver

# Windows x64
GOOS=windows GOARCH=amd64 go build -o bin/printserver-windows-amd64.exe ./cmd/printserver

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o bin/printserver-darwin-amd64 ./cmd/printserver

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o bin/printserver-darwin-arm64 ./cmd/printserver

# Or build all at once
make build-all
```

### Binary Sizes

| Platform | Size |
|----------|------|
| Windows x64 | ~8.3 MB |
| Linux ARM64 | ~7.8 MB |
| Linux x64 | ~8.0 MB |

---

## Deployment

### Raspberry Pi Zero 2 W

**Hardware Required:**
- Raspberry Pi Zero 2 W ($15)
- microSD card 8GB+ ($8)
- USB power supply 5V 2.5A ($8)
- USB OTG adapter (for USB printers) ($3)

**Installation:**

```bash
# 1. Copy binary to Pi
scp bin/printserver-linux-arm64 pi@raspberrypi:~/printserver

# 2. SSH into Pi
ssh pi@raspberrypi

# 3. Make executable
chmod +x printserver

# 4. Create config file
nano config.yaml

# 5. Run
./printserver
```

**Run as systemd service:**

Create `/etc/systemd/system/printserver.service`:

```ini
[Unit]
Description=JetSetGo Print Server
After=network.target

[Service]
Type=simple
User=pi
WorkingDirectory=/home/pi
ExecStart=/home/pi/printserver
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable printserver
sudo systemctl start printserver
sudo systemctl status printserver
```

### Windows

1. Download `printserver-windows-amd64.exe`
2. Create `config.yaml` in same directory
3. Run the executable
4. (Optional) Create a Windows Service using NSSM

### Docker

```dockerfile
FROM alpine:latest
COPY printserver-linux-amd64 /app/printserver
COPY config.yaml /app/config.yaml
WORKDIR /app
EXPOSE 8080
CMD ["./printserver"]
```

```bash
docker build -t jetsetgo-printserver .
docker run -d -p 8080:8080 jetsetgo-printserver
```

---

## Troubleshooting

### Server won't start

```bash
# Check if port is in use
netstat -an | grep 8080

# Check config file syntax
cat config.yaml
```

### Printer not connecting

```bash
# Test network connectivity
ping 192.168.1.100

# Test printer port
nc -zv 192.168.1.100 9100

# Check printer status in web UI
curl http://localhost:8080/api/printers
```

### Test print fails

```bash
# Check emulator is running
docker ps | grep escpos

# Check emulator logs
docker logs escpos-emulator

# View raw ESC/POS output
curl -X POST http://localhost:8080/api/printers/emulator-1/test
```

---

## Next Steps

- [ ] Implement cloud polling for print jobs
- [ ] Add USB printer support
- [ ] Build cloud ESC/POS generator (FastAPI)
- [ ] Add server registration flow
- [ ] Create installer packages (.deb, .msi)
- [ ] Build pre-configured Raspberry Pi SD card image

---

## References

- [PRD.md](./PRD.md) - Product Requirements Document
- [ESC/POS Command Reference](https://reference.epson-biz.com/modules/ref_escpos/index.php)
- [Go Documentation](https://go.dev/doc/)
- [Raspberry Pi Zero 2 W](https://www.raspberrypi.com/products/raspberry-pi-zero-2-w/)
- [escpos-netprinter](https://github.com/gilbertfl/escpos-netprinter) - Docker ESC/POS emulator

# JetSetGo Local Print Server

A lightweight print server that connects thermal printers to the JetSetGo cloud platform.

## Quick Start

```bash
# Build
go build -o printserver ./cmd/printserver

# Run
./printserver
```

Then open http://localhost:8080 in your browser.

## Features

- **Network Printer Support** - Connect to ESC/POS printers via TCP (port 9100)
- **USB Printer Support** - Coming soon
- **Auto-Discovery** - Scan local network for printers
- **Web UI** - Simple configuration interface
- **Cloud Integration** - Polls JetSetGo cloud for print jobs

## Configuration

Copy `configs/config.example.yaml` to `config.yaml` and edit:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

cloud:
  endpoint: "https://api.jetsetgo.world/api/v1/print"
  server_id: "your-server-id"
  api_key: "your-api-key"
  poll_interval: 2s

printers:
  - id: "receipt-1"
    name: "Front Desk"
    type: "network"
    address: "192.168.1.100"
    port: 9100
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/printers` | GET | List configured printers |
| `/api/printers/discover` | POST | Scan for printers |
| `/api/printers/{id}/test` | POST | Send test print |
| `/api/print` | POST | Print ESC/POS data |
| `/api/status` | GET | Server status |

## Cross-Compilation

Build for all platforms:

```bash
# Raspberry Pi Zero 2 W / Pi 3/4/5 (ARM64)
GOOS=linux GOARCH=arm64 go build -o printserver-arm64 ./cmd/printserver

# Original Pi Zero W (ARM v6)
GOOS=linux GOARCH=arm GOARM=6 go build -o printserver-arm ./cmd/printserver

# Linux x64
GOOS=linux GOARCH=amd64 go build -o printserver-linux ./cmd/printserver

# Windows
GOOS=windows GOARCH=amd64 go build -o printserver.exe ./cmd/printserver
```

## Deployment on Raspberry Pi

1. Download the ARM64 binary
2. Copy to Pi: `scp printserver-arm64 pi@raspberrypi:~/printserver`
3. Make executable: `chmod +x printserver`
4. Create config file
5. Run: `./printserver`

### Run as systemd service

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
```

## Development

```bash
# Install dependencies
go mod tidy

# Run in development
go run ./cmd/printserver

# Run tests
go test ./...

# Build all platforms
make build-all
```

## License

Proprietary - JetSetGo

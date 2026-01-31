.PHONY: build run clean test build-all

# Binary name
BINARY=printserver

# Build for current platform
build:
	go build -o bin/$(BINARY) ./cmd/printserver

# Run the server
run: build
	./bin/$(BINARY)

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests
test:
	go test -v ./...

# Build for all platforms
build-all: clean
	mkdir -p bin
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY)-linux-amd64 ./cmd/printserver
	# Linux ARM64 (Pi Zero 2 W, Pi 3/4/5)
	GOOS=linux GOARCH=arm64 go build -o bin/$(BINARY)-linux-arm64 ./cmd/printserver
	# Linux ARM (Pi Zero W original)
	GOOS=linux GOARCH=arm GOARM=6 go build -o bin/$(BINARY)-linux-arm ./cmd/printserver
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY)-windows-amd64.exe ./cmd/printserver
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY)-darwin-amd64 ./cmd/printserver
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -o bin/$(BINARY)-darwin-arm64 ./cmd/printserver

# Show binary sizes
sizes:
	ls -lh bin/

# Development mode with hot reload (requires air)
dev:
	air

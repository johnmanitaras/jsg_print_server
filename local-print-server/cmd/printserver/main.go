package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jetsetgo/local-print-server/internal/api"
	"github.com/jetsetgo/local-print-server/internal/config"
)

func main() {
	fmt.Println("JetSetGo Local Print Server")
	fmt.Println("===========================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: Could not load config file: %v", err)
		log.Println("Using default configuration")
		cfg = config.Default()
	}

	// Print configuration
	fmt.Printf("Server Port: %d\n", cfg.Server.Port)
	fmt.Printf("Cloud Endpoint: %s\n", cfg.Cloud.Endpoint)

	// Start the HTTP server
	server := api.NewServer(cfg)

	fmt.Printf("\nStarting server on http://localhost:%d\n", cfg.Server.Port)
	fmt.Println("Press Ctrl+C to stop")

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
		os.Exit(1)
	}
}

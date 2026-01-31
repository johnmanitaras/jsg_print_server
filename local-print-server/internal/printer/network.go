package printer

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// NetworkPrinter represents a network-connected thermal printer
type NetworkPrinter struct {
	id      string
	name    string
	address string
	port    int
	conn    net.Conn
	mu      sync.Mutex
}

// NewNetworkPrinter creates a new network printer
func NewNetworkPrinter(id, name, address string, port int) *NetworkPrinter {
	return &NetworkPrinter{
		id:      id,
		name:    name,
		address: address,
		port:    port,
	}
}

// ID returns the printer ID
func (p *NetworkPrinter) ID() string {
	return p.id
}

// Name returns the printer name
func (p *NetworkPrinter) Name() string {
	return p.name
}

// Type returns the printer type
func (p *NetworkPrinter) Type() string {
	return "network"
}

// Status returns the printer status
func (p *NetworkPrinter) Status() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Try to connect to check status
	addr := fmt.Sprintf("%s:%d", p.address, p.port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return "offline"
	}
	conn.Close()
	return "online"
}

// Print sends data to the printer
func (p *NetworkPrinter) Print(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", p.address, p.port)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to printer: %w", err)
	}
	defer conn.Close()

	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send data to printer: %w", err)
	}

	return nil
}

// Close closes the printer connection
func (p *NetworkPrinter) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		return err
	}
	return nil
}

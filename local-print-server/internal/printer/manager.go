package printer

import (
	"errors"
	"fmt"
	"net"
	"time"
)

// Manager manages printer connections and print jobs
type Manager struct {
	printers map[string]Printer
}

// Printer represents a thermal printer
type Printer interface {
	ID() string
	Name() string
	Type() string
	Status() string
	Print(data []byte) error
	Close() error
}

// DiscoveredPrinter represents a discovered printer
type DiscoveredPrinter struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Address  string `json:"address,omitempty"`
	Port     int    `json:"port,omitempty"`
	VendorID string `json:"vendor_id,omitempty"`
}

// NewManager creates a new printer manager
func NewManager() *Manager {
	return &Manager{
		printers: make(map[string]Printer),
	}
}

// AddPrinter adds a printer to the manager
func (m *Manager) AddPrinter(p Printer) {
	m.printers[p.ID()] = p
}

// GetPrinter gets a printer by ID
func (m *Manager) GetPrinter(id string) (Printer, error) {
	p, ok := m.printers[id]
	if !ok {
		return nil, errors.New("printer not found: " + id)
	}
	return p, nil
}

// Print sends data to a printer
func (m *Manager) Print(printerID string, data []byte) error {
	p, err := m.GetPrinter(printerID)
	if err != nil {
		return err
	}
	return p.Print(data)
}

// TestPrint sends a test print to a printer
func (m *Manager) TestPrint(printerID string) error {
	p, err := m.GetPrinter(printerID)
	if err != nil {
		return err
	}

	// ESC/POS test receipt
	testData := buildTestReceipt()
	return p.Print(testData)
}

// Discover scans for available printers
func (m *Manager) Discover() ([]DiscoveredPrinter, error) {
	discovered := make([]DiscoveredPrinter, 0)

	// Discover network printers on common ports
	networkPrinters := discoverNetworkPrinters()
	discovered = append(discovered, networkPrinters...)

	// TODO: Discover USB printers
	// This requires platform-specific code or CGO with libusb

	return discovered, nil
}

// discoverNetworkPrinters scans for network printers on port 9100
func discoverNetworkPrinters() []DiscoveredPrinter {
	discovered := make([]DiscoveredPrinter, 0)

	// Common local network ranges
	// In production, this should be configurable or use mDNS/Bonjour
	subnets := []string{"192.168.1.", "192.168.0.", "10.0.0."}

	for _, subnet := range subnets {
		for i := 1; i <= 254; i++ {
			ip := fmt.Sprintf("%s%d", subnet, i)
			if isPortOpen(ip, 9100, 100*time.Millisecond) {
				discovered = append(discovered, DiscoveredPrinter{
					ID:      fmt.Sprintf("network-%s", ip),
					Name:    fmt.Sprintf("Printer at %s", ip),
					Type:    "network",
					Address: ip,
					Port:    9100,
				})
			}
		}
	}

	return discovered
}

// isPortOpen checks if a port is open on a host
func isPortOpen(host string, port int, timeout time.Duration) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// buildTestReceipt creates ESC/POS commands for a test receipt
func buildTestReceipt() []byte {
	var data []byte

	// Initialize printer
	data = append(data, 0x1B, 0x40) // ESC @

	// Center align
	data = append(data, 0x1B, 0x61, 0x01) // ESC a 1

	// Bold on
	data = append(data, 0x1B, 0x45, 0x01) // ESC E 1

	// Double size
	data = append(data, 0x1D, 0x21, 0x11) // GS ! 0x11

	data = append(data, []byte("JETSETGO\n")...)

	// Normal size
	data = append(data, 0x1D, 0x21, 0x00) // GS ! 0x00

	// Bold off
	data = append(data, 0x1B, 0x45, 0x00) // ESC E 0

	data = append(data, []byte("Print Server\n")...)
	data = append(data, []byte("-------------------\n")...)
	data = append(data, []byte("\n")...)

	// Left align
	data = append(data, 0x1B, 0x61, 0x00) // ESC a 0

	data = append(data, []byte("Test Print\n")...)
	data = append(data, []byte(fmt.Sprintf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05")))...)
	data = append(data, []byte("\n")...)

	// Center align
	data = append(data, 0x1B, 0x61, 0x01) // ESC a 1

	data = append(data, []byte("-------------------\n")...)
	data = append(data, []byte("Printer OK!\n")...)
	data = append(data, []byte("\n\n\n")...)

	// Cut paper (partial cut)
	data = append(data, 0x1D, 0x56, 0x42, 0x00) // GS V 66 0

	return data
}

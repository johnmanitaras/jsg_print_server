package cloud

import "time"

// ConnectionStatus represents the cloud connection status
type ConnectionStatus struct {
	Connected    bool
	Reconnecting bool
	LastError    string
	LastSeen     time.Time
}

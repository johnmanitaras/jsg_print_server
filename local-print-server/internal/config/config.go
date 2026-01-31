package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Cloud    CloudConfig    `yaml:"cloud"`
	Printers []PrinterConfig `yaml:"printers"`
}

// ServerConfig represents the local server configuration
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// CloudConfig represents the cloud server connection configuration
type CloudConfig struct {
	Endpoint     string        `yaml:"endpoint"`
	ServerID     string        `yaml:"server_id"`
	APIKey       string        `yaml:"api_key"`
	PollInterval time.Duration `yaml:"poll_interval"`
}

// PrinterConfig represents a printer configuration
type PrinterConfig struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	Type      string `yaml:"type"` // "usb" or "network"
	VendorID  string `yaml:"vendor_id,omitempty"`
	ProductID string `yaml:"product_id,omitempty"`
	Address   string `yaml:"address,omitempty"`
	Port      int    `yaml:"port,omitempty"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Cloud: CloudConfig{
			Endpoint:     "https://api.jetsetgo.world/api/v1/print",
			PollInterval: 2 * time.Second,
		},
		Printers: []PrinterConfig{},
	}
}

// Load loads configuration from the config file
func Load() (*Config, error) {
	// Try to find config file in common locations
	configPaths := []string{
		"config.yaml",
		"configs/config.yaml",
		"/etc/printserver/config.yaml",
	}

	var data []byte
	var err error

	for _, path := range configPaths {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

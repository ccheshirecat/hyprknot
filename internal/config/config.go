package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server ServerConfig `yaml:"server"`
	Knot   KnotConfig   `yaml:"knot"`
	Auth   AuthConfig   `yaml:"auth"`
	Log    LogConfig    `yaml:"log"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
	IdleTimeout  int    `yaml:"idle_timeout"`
}

// KnotConfig contains KnotDNS configuration
type KnotConfig struct {
	ConfigPath   string   `yaml:"config_path"`
	SocketPath   string   `yaml:"socket_path"`
	KnotcPath    string   `yaml:"knotc_path"`
	AllowedZones []string `yaml:"allowed_zones"`
	DataDir      string   `yaml:"data_dir"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled bool     `yaml:"enabled"`
	APIKeys []string `yaml:"api_keys"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "127.0.0.1",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
		},
		Knot: KnotConfig{
			ConfigPath:   "/etc/knot/knot.conf",
			SocketPath:   "/run/knot/knot.sock",
			KnotcPath:    "/usr/sbin/knotc", // Default for Debian/Ubuntu
			AllowedZones: []string{},
			DataDir:      "/var/lib/knot",
		},
		Auth: AuthConfig{
			Enabled: true,
			APIKeys: []string{},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}

// LoadConfig loads configuration from file or returns default
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		return config, nil
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate knot config
	if c.Knot.KnotcPath == "" {
		return fmt.Errorf("knotc_path cannot be empty")
	}

	// Check if knotc exists
	if _, err := os.Stat(c.Knot.KnotcPath); os.IsNotExist(err) {
		return fmt.Errorf("knotc binary not found at: %s", c.Knot.KnotcPath)
	}

	// Validate log level
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLevels[c.Log.Level] {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	return nil
}

// SaveConfig saves configuration to file
func (c *Config) SaveConfig(configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetAddress returns the server address
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

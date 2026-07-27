package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Version int `toml:"version"`

	ListenAddress string `toml:"listen_address"`
	TargetHost    string `toml:"target_host"`

	Domains []string `toml:"domains"`

	LogDirectory string `toml:"log_directory"`

	MinimumPort int `toml:"minimum_port"`
	MaximumPort int `toml:"maximum_port"`

	ReadHeaderTimeout string `toml:"read_header_timeout"`
	IdleTimeout       string `toml:"idle_timeout"`
	ShutdownTimeout   string `toml:"shutdown_timeout"`

	PreserveOriginalHost  bool `toml:"preserve_original_host"`
	TrustForwardedHeaders bool `toml:"trust_forwarded_headers"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing TOML: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if len(c.Domains) == 0 {
		return fmt.Errorf("no domains configured")
	}

	for _, d := range c.Domains {
		if strings.TrimSpace(d) == "" {
			return fmt.Errorf("empty domain in list")
		}
	}

	if c.ListenAddress == "" {
		return fmt.Errorf("listen_address is required")
	}

	if c.LogDirectory == "" {
		return fmt.Errorf("log_directory is required")
	}

	if c.MinimumPort <= 0 || c.MinimumPort > 65535 {
		return fmt.Errorf("minimum_port must be between 1 and 65535")
	}

	if c.MaximumPort <= 0 || c.MaximumPort > 65535 {
		return fmt.Errorf("maximum_port must be between 1 and 65535")
	}

	if c.MinimumPort > c.MaximumPort {
		return fmt.Errorf("minimum_port cannot exceed maximum_port")
	}

	if c.TargetHost == "" {
		c.TargetHost = "127.0.0.1"
	} else if c.TargetHost != "127.0.0.1" && c.TargetHost != "localhost" {
		return fmt.Errorf("target_host must be 127.0.0.1 or localhost")
	}

	return nil
}

func (c *Config) GetReadHeaderTimeout() time.Duration {
	if c.ReadHeaderTimeout == "" {
		return 10 * time.Second
	}
	d, err := time.ParseDuration(c.ReadHeaderTimeout)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

func (c *Config) GetIdleTimeout() time.Duration {
	if c.IdleTimeout == "" {
		return 120 * time.Second
	}
	d, err := time.ParseDuration(c.IdleTimeout)
	if err != nil {
		return 120 * time.Second
	}
	return d
}

func (c *Config) GetShutdownTimeout() time.Duration {
	if c.ShutdownTimeout == "" {
		return 15 * time.Second
	}
	d, err := time.ParseDuration(c.ShutdownTimeout)
	if err != nil {
		return 15 * time.Second
	}
	return d
}

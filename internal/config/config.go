package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	RefreshRate float64  `yaml:"refresh_rate"` // Times per second to check IP
	SyncRate    float64  `yaml:"sync_rate"`    // Times per minute to verify DNS
	Domains     []Domain `yaml:"domains"`
}

// Domain represents a domain configuration
type Domain struct {
	ZoneName string   `yaml:"zone_name"`
	Records  []Record `yaml:"records"`
}

// Record represents a DNS record configuration
type Record struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"` // A or AAAA
	Proxied bool   `yaml:"proxied"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.RefreshRate <= 0 {
		return fmt.Errorf("refresh_rate must be greater than 0")
	}
	if c.SyncRate <= 0 {
		return fmt.Errorf("sync_rate must be greater than 0")
	}
	if len(c.Domains) == 0 {
		return fmt.Errorf("at least one domain must be configured")
	}

	for i, domain := range c.Domains {
		if domain.ZoneName == "" {
			return fmt.Errorf("domain %d: zone_name is required", i)
		}
		if len(domain.Records) == 0 {
			return fmt.Errorf("domain %s: at least one record must be configured", domain.ZoneName)
		}

		for j, record := range domain.Records {
			if record.Name == "" {
				return fmt.Errorf("domain %s, record %d: name is required", domain.ZoneName, j)
			}
			if record.Type != "A" && record.Type != "AAAA" {
				return fmt.Errorf("domain %s, record %s: type must be A or AAAA", domain.ZoneName, record.Name)
			}
		}
	}

	return nil
}

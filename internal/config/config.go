package config

import (
	"fmt"
	"math"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	RefreshRate  float64  `yaml:"refresh_rate"` // Times per second to check IP
	SyncRate     float64  `yaml:"sync_rate"`    // Times per minute to verify DNS
	SupportsIPv6 bool     `yaml:"supports_ipv6"`
	Domains      []Domain `yaml:"domains"`
}

// Domain represents a domain configuration
type Domain struct {
	ZoneName string   `yaml:"zone_name"`
	Provider string   `yaml:"provider"` // cloudflare or route53
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
	if math.IsNaN(c.RefreshRate) || math.IsInf(c.RefreshRate, 0) {
		return fmt.Errorf("refresh_rate must be a finite number")
	}
	if c.RefreshRate <= 0 {
		return fmt.Errorf("refresh_rate must be greater than 0")
	}
	if time.Duration(float64(time.Second)/c.RefreshRate) <= 0 {
		return fmt.Errorf("refresh_rate is too high and results in an invalid interval")
	}

	if math.IsNaN(c.SyncRate) || math.IsInf(c.SyncRate, 0) {
		return fmt.Errorf("sync_rate must be a finite number")
	}
	if c.SyncRate <= 0 {
		return fmt.Errorf("sync_rate must be greater than 0")
	}
	if time.Duration(float64(time.Minute)/c.SyncRate) <= 0 {
		return fmt.Errorf("sync_rate is too high and results in an invalid interval")
	}

	if len(c.Domains) == 0 {
		return fmt.Errorf("at least one domain must be configured")
	}

	for i, domain := range c.Domains {
		if domain.ZoneName == "" {
			return fmt.Errorf("domain %d: zone_name is required", i)
		}
		if domain.Provider == "" {
			domain.Provider = "cloudflare"
			c.Domains[i].Provider = "cloudflare" // Default to cloudflare
		}
		if domain.Provider != "cloudflare" && domain.Provider != "route53" {
			return fmt.Errorf("domain %s: unsupported provider %s", domain.ZoneName, domain.Provider)
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
			if record.Type == "AAAA" && !c.SupportsIPv6 {
				return fmt.Errorf("domain %s, record %s: AAAA record configured but supports_ipv6 is false", domain.ZoneName, record.Name)
			}
		}
	}

	return nil
}

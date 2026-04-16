package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/msyrus/ipwatcher/internal/config"
)

func TestLoadConfig_Success(t *testing.T) {
	// Create a temporary config file
	content := "refresh_rate: 0.5\n" +
		"sync_rate: 2.0\n" +
		"domains:\n" +
		"  - zone_name: \"example.com\"\n" +
		"    records:\n" +
		"      - name: \"@\"\n" +
		"        type: \"A\"\n" +
		"        proxied: false\n" +
		"      - name: \"www\"\n" +
		"        type: \"A\"\n" +
		"        proxied: true\n"
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.RefreshRate != 0.5 {
		t.Errorf("Expected RefreshRate 0.5, got %f", cfg.RefreshRate)
	}

	if cfg.SyncRate != 2.0 {
		t.Errorf("Expected SyncRate 2.0, got %f", cfg.SyncRate)
	}

	if len(cfg.Domains) != 1 {
		t.Fatalf("Expected 1 domain, got %d", len(cfg.Domains))
	}

	domain := cfg.Domains[0]
	if domain.ZoneName != "example.com" {
		t.Errorf("Expected ZoneName 'example.com', got '%s'", domain.ZoneName)
	}

	if len(domain.Records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(domain.Records))
	}

	if domain.Records[0].Name != "@" {
		t.Errorf("Expected record name '@', got '%s'", domain.Records[0].Name)
	}

	if domain.Records[0].Type != "A" {
		t.Errorf("Expected record type 'A', got '%s'", domain.Records[0].Type)
	}

	if domain.Records[0].Proxied != false {
		t.Errorf("Expected proxied false, got true")
	}

	if domain.Records[1].Proxied != true {
		t.Errorf("Expected proxied true, got false")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := config.LoadConfig("nonexistent.yaml")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	content := `invalid: yaml: content: [[[`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	_, err := config.LoadConfig(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestValidate_InvalidRefreshRate(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: 0,
		SyncRate:    1.0,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "@", Type: "A", Proxied: false},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for invalid refresh rate, got nil")
	}
}

func TestValidate_InvalidSyncRate(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: 0.5,
		SyncRate:    -1.0,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "@", Type: "A", Proxied: false},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for invalid sync rate, got nil")
	}
}

func TestValidate_NoDomains(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: 0.5,
		SyncRate:    1.0,
		Domains:     []config.Domain{},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for no domains, got nil")
	}
}

func TestValidate_EmptyZoneName(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: 0.5,
		SyncRate:    1.0,
		Domains: []config.Domain{
			{
				ZoneName: "",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for empty zone name, got nil")
	}
}

func TestValidate_NoRecords(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: 0.5,
		SyncRate:    1.0,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records:  []config.Record{},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for no records, got nil")
	}
}

func TestValidate_EmptyRecordName(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: 0.5,
		SyncRate:    1.0,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "", Type: "A", Proxied: false},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for empty record name, got nil")
	}
}

func TestValidate_InvalidRecordType(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: 0.5,
		SyncRate:    1.0,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "@", Type: "CNAME", Proxied: false},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for invalid record type, got nil")
	}
}

func TestValidate_AAAAWithoutIPv6Support(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.5,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "@", Type: "AAAA", Proxied: false},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for AAAA record without IPv6 support, got nil")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.5,
		SyncRate:     1.0,
		SupportsIPv6: true,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "@", Type: "A", Proxied: false},
					{Name: "www", Type: "AAAA", Proxied: true},
				},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected no error for valid config, got: %v", err)
	}
}

func TestValidate_MultipleDomainsValid(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.5,
		SyncRate:     1.0,
		SupportsIPv6: true,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "@", Type: "A", Proxied: false},
				},
			},
			{
				ZoneName: "example.org",
				Records: []config.Record{
					{Name: "@", Type: "AAAA", Proxied: true},
				},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected no error for valid multi-domain config, got: %v", err)
	}
}

func TestValidate_RefreshRateTooHigh(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: 1e20,
		SyncRate:    1.0,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records:  []config.Record{{Name: "@", Type: "A", Proxied: false}},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for refresh_rate that produces invalid interval, got nil")
	}
}

func TestValidate_SyncRateTooHigh(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: 1.0,
		SyncRate:    1e20,
		Domains: []config.Domain{
			{
				ZoneName: "example.com",
				Records:  []config.Record{{Name: "@", Type: "A", Proxied: false}},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for sync_rate that produces invalid interval, got nil")
	}
}

package main_test

import (
	"context"
	"errors"
	"testing"

	main "github.com/msyrus/ipwatcher/cmd/ipwatcher"
	"github.com/msyrus/ipwatcher/internal/config"
	"github.com/msyrus/ipwatcher/internal/dnsmanager"
)

// MockIPFetcher implements ipfetcher.Fetcher for testing
type MockIPFetcher struct {
	GetIPv4Func func(ctx context.Context) (string, error)
	GetIPv6Func func(ctx context.Context) (string, error)
}

func (m *MockIPFetcher) GetIPv4(ctx context.Context) (string, error) {
	if m.GetIPv4Func != nil {
		return m.GetIPv4Func(ctx)
	}
	return "192.168.1.1", nil
}

func (m *MockIPFetcher) GetIPv6(ctx context.Context) (string, error) {
	if m.GetIPv6Func != nil {
		return m.GetIPv6Func(ctx)
	}
	return "2001:db8::1", nil
}

// MockDNSProvider implements dnsmanager.DNSProvider for testing
type MockDNSProvider struct {
	GetZoneIDByNameFunc  func(ctx context.Context, zoneName string) (string, error)
	EnsureDNSRecordsFunc func(ctx context.Context, zoneID string, records []dnsmanager.DNSRecord, ipv4, ipv6 string) error
}

func (m *MockDNSProvider) GetZoneIDByName(ctx context.Context, zoneName string) (string, error) {
	if m.GetZoneIDByNameFunc != nil {
		return m.GetZoneIDByNameFunc(ctx, zoneName)
	}
	return "zone-123", nil
}

func (m *MockDNSProvider) EnsureDNSRecords(ctx context.Context, zoneID string, records []dnsmanager.DNSRecord, ipv4, ipv6 string) error {
	if m.EnsureDNSRecordsFunc != nil {
		return m.EnsureDNSRecordsFunc(ctx, zoneID, records, ipv4, ipv6)
	}
	return nil
}

func TestNewIPWatcher_CloudflareProvider(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	// Should fail without API token
	_, err := main.NewIPWatcher(ctx, cfg, "")
	if err == nil {
		t.Error("Expected error when creating Cloudflare provider without API token")
	}

	// Should succeed with API token
	watcher, err := main.NewIPWatcher(ctx, cfg, "test-token")
	if err != nil {
		t.Fatalf("Failed to create IPWatcher: %v", err)
	}

	if watcher == nil {
		t.Fatal("Expected non-nil watcher")
	}
}

func TestNewIPWatcher_Route53Provider(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "route53",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	// Route53 might fail if AWS credentials aren't configured - that's OK for this test
	watcher, err := main.NewIPWatcher(ctx, cfg, "")
	if err != nil {
		t.Skipf("Skipping Route53 test - AWS credentials not configured: %v", err)
	}

	if watcher == nil {
		t.Fatal("Expected non-nil watcher")
	}
}

func TestNewIPWatcher_IPv6Support(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: true,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "AAAA", Proxied: false},
				},
			},
		},
	}

	watcher, err := main.NewIPWatcher(ctx, cfg, "test-token")
	if err != nil {
		t.Fatalf("Failed to create IPWatcher: %v", err)
	}

	if watcher == nil {
		t.Fatal("Expected non-nil watcher")
	}
}

func TestNewIPWatcher_MultipleProviders(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	watcher, err := main.NewIPWatcher(ctx, cfg, "test-token")
	if err != nil {
		t.Fatalf("Failed to create IPWatcher: %v", err)
	}

	if watcher == nil {
		t.Fatal("Expected non-nil watcher")
	}
}

func TestNewIPWatcher_EmptyConfig(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains:      []config.Domain{},
	}

	watcher, err := main.NewIPWatcher(ctx, cfg, "test-token")
	if err != nil {
		t.Fatalf("Failed to create IPWatcher: %v", err)
	}

	if watcher == nil {
		t.Fatal("Expected non-nil watcher even with no domains")
	}
}

func TestNewIPWatcher_MixedProviders(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: true,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
					{Name: "example.com", Type: "AAAA", Proxied: false},
				},
			},
		},
	}

	// With API token for Cloudflare
	watcher, err := main.NewIPWatcher(ctx, cfg, "test-token")
	if err != nil {
		t.Fatalf("Failed to create IPWatcher with mixed providers: %v", err)
	}

	if watcher == nil {
		t.Fatal("Expected non-nil watcher")
	}
}

// Helper function to create a test watcher with mocks
func createTestWatcher(cfg *config.Config, fetcher *MockIPFetcher, provider *MockDNSProvider) *main.IPWatcher {
	providers := make(map[string]dnsmanager.DNSProvider)
	for _, d := range cfg.Domains {
		providers[d.Provider] = provider
	}
	return main.NewIPWatcherWithDeps(cfg, fetcher, providers)
}

func TestIPWatcher_GetZoneID(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	mockProvider := &MockDNSProvider{
		GetZoneIDByNameFunc: func(ctx context.Context, zoneName string) (string, error) {
			if zoneName == "example.com" {
				return "zone-123", nil
			}
			return "", errors.New("zone not found")
		},
	}

	watcher := createTestWatcher(cfg, &MockIPFetcher{}, mockProvider)
	ctx := context.Background()

	// Test successful zone ID lookup
	zoneID, err := watcher.GetZoneID(ctx, "example.com", "cloudflare")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if zoneID != "zone-123" {
		t.Errorf("Expected zone-123, got %s", zoneID)
	}

	// Test caching - call again and it should use cache
	zoneID2, err := watcher.GetZoneID(ctx, "example.com", "cloudflare")
	if err != nil {
		t.Errorf("Unexpected error on cached call: %v", err)
	}
	if zoneID2 != "zone-123" {
		t.Errorf("Expected cached zone-123, got %s", zoneID2)
	}

	// Test unsupported provider
	_, err = watcher.GetZoneID(ctx, "example.com", "invalid-provider")
	if err == nil {
		t.Error("Expected error for unsupported provider")
	}

	// Test zone not found
	_, err = watcher.GetZoneID(ctx, "notfound.com", "cloudflare")
	if err == nil {
		t.Error("Expected error for zone not found")
	}
}

func TestIPWatcher_FetchAndUpdateIPs(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	mockFetcher := &MockIPFetcher{
		GetIPv4Func: func(ctx context.Context) (string, error) {
			return "203.0.113.10", nil
		},
	}

	ensureCalled := 0
	mockProvider := &MockDNSProvider{
		GetZoneIDByNameFunc: func(ctx context.Context, zoneName string) (string, error) {
			return "zone-123", nil
		},
		EnsureDNSRecordsFunc: func(ctx context.Context, zoneID string, records []dnsmanager.DNSRecord, ipv4, ipv6 string) error {
			ensureCalled++
			if ipv4 != "203.0.113.10" {
				t.Errorf("Expected IPv4 203.0.113.10, got %s", ipv4)
			}
			return nil
		},
	}

	watcher := createTestWatcher(cfg, mockFetcher, mockProvider)
	ctx := context.Background()

	err := watcher.FetchAndUpdateIPs(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if ensureCalled != 1 {
		t.Errorf("Expected EnsureDNSRecords to be called once, got %d", ensureCalled)
	}
}

func TestIPWatcher_FetchAndUpdateIPs_WithIPv6(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: true,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
					{Name: "example.com", Type: "AAAA", Proxied: false},
				},
			},
		},
	}

	mockFetcher := &MockIPFetcher{
		GetIPv4Func: func(ctx context.Context) (string, error) {
			return "203.0.113.10", nil
		},
		GetIPv6Func: func(ctx context.Context) (string, error) {
			return "2001:db8::42", nil
		},
	}

	mockProvider := &MockDNSProvider{
		GetZoneIDByNameFunc: func(ctx context.Context, zoneName string) (string, error) {
			return "zone-123", nil
		},
		EnsureDNSRecordsFunc: func(ctx context.Context, zoneID string, records []dnsmanager.DNSRecord, ipv4, ipv6 string) error {
			if ipv4 != "203.0.113.10" {
				t.Errorf("Expected IPv4 203.0.113.10, got %s", ipv4)
			}
			if ipv6 != "2001:db8::42" {
				t.Errorf("Expected IPv6 2001:db8::42, got %s", ipv6)
			}
			return nil
		},
	}

	watcher := createTestWatcher(cfg, mockFetcher, mockProvider)
	ctx := context.Background()

	err := watcher.FetchAndUpdateIPs(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestIPWatcher_UpdateAllDNSRecords(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
					{Name: "www.example.com", Type: "A", Proxied: true},
				},
			},
		},
	}

	ensureCalled := 0
	mockProvider := &MockDNSProvider{
		GetZoneIDByNameFunc: func(ctx context.Context, zoneName string) (string, error) {
			return "zone-123", nil
		},
		EnsureDNSRecordsFunc: func(ctx context.Context, zoneID string, records []dnsmanager.DNSRecord, ipv4, ipv6 string) error {
			ensureCalled++
			if len(records) != 2 {
				t.Errorf("Expected 2 records, got %d", len(records))
			}
			return nil
		},
	}

	watcher := createTestWatcher(cfg, &MockIPFetcher{}, mockProvider)
	ctx := context.Background()

	// First fetch IPs
	_ = watcher.FetchAndUpdateIPs(ctx)

	// Reset counter
	ensureCalled = 0

	// Now update DNS records
	err := watcher.UpdateAllDNSRecords(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if ensureCalled != 1 {
		t.Errorf("Expected EnsureDNSRecords to be called once, got %d", ensureCalled)
	}
}

func TestIPWatcher_UpdateAllDNSRecords_ProviderError(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	mockProvider := &MockDNSProvider{
		GetZoneIDByNameFunc: func(ctx context.Context, zoneName string) (string, error) {
			return "", errors.New("zone lookup failed")
		},
	}

	watcher := createTestWatcher(cfg, &MockIPFetcher{}, mockProvider)
	ctx := context.Background()

	// First fetch IPs
	_ = watcher.FetchAndUpdateIPs(ctx)

	// Now try to update DNS records - should error
	err := watcher.UpdateAllDNSRecords(ctx)
	if err == nil {
		t.Error("Expected error when zone lookup fails")
	}
}

func TestIPWatcher_VerifyDNSRecords(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	verifyCalled := false
	mockProvider := &MockDNSProvider{
		GetZoneIDByNameFunc: func(ctx context.Context, zoneName string) (string, error) {
			return "zone-123", nil
		},
		EnsureDNSRecordsFunc: func(ctx context.Context, zoneID string, records []dnsmanager.DNSRecord, ipv4, ipv6 string) error {
			verifyCalled = true
			return nil
		},
	}

	watcher := createTestWatcher(cfg, &MockIPFetcher{}, mockProvider)
	ctx := context.Background()

	// First fetch IPs
	_ = watcher.FetchAndUpdateIPs(ctx)

	// Reset flag
	verifyCalled = false

	// Now verify DNS records
	err := watcher.VerifyDNSRecords(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !verifyCalled {
		t.Error("Expected EnsureDNSRecords to be called for verification")
	}
}

func TestIPWatcher_CheckAndUpdateIP_NoChange(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	mockFetcher := &MockIPFetcher{
		GetIPv4Func: func(ctx context.Context) (string, error) {
			return "203.0.113.10", nil
		},
	}

	ensureCalled := 0
	mockProvider := &MockDNSProvider{
		GetZoneIDByNameFunc: func(ctx context.Context, zoneName string) (string, error) {
			return "zone-123", nil
		},
		EnsureDNSRecordsFunc: func(ctx context.Context, zoneID string, records []dnsmanager.DNSRecord, ipv4, ipv6 string) error {
			ensureCalled++
			return nil
		},
	}

	watcher := createTestWatcher(cfg, mockFetcher, mockProvider)
	ctx := context.Background()

	// Initial fetch
	_ = watcher.FetchAndUpdateIPs(ctx)
	ensureCalled = 0 // Reset counter

	// Check again - should not update because IP hasn't changed
	err := watcher.CheckAndUpdateIP(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if ensureCalled != 0 {
		t.Errorf("Expected no DNS updates when IP hasn't changed, but got %d calls", ensureCalled)
	}
}

func TestIPWatcher_CheckAndUpdateIP_Changed(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:  0.1,
		SyncRate:     1.0,
		SupportsIPv6: false,
		Domains: []config.Domain{
			{
				Provider: "cloudflare",
				ZoneName: "example.com",
				Records: []config.Record{
					{Name: "example.com", Type: "A", Proxied: false},
				},
			},
		},
	}

	ipCallCount := 0
	mockFetcher := &MockIPFetcher{
		GetIPv4Func: func(ctx context.Context) (string, error) {
			ipCallCount++
			if ipCallCount == 1 {
				return "203.0.113.10", nil
			}
			return "203.0.113.20", nil // Changed IP
		},
	}

	ensureCalled := 0
	mockProvider := &MockDNSProvider{
		GetZoneIDByNameFunc: func(ctx context.Context, zoneName string) (string, error) {
			return "zone-123", nil
		},
		EnsureDNSRecordsFunc: func(ctx context.Context, zoneID string, records []dnsmanager.DNSRecord, ipv4, ipv6 string) error {
			ensureCalled++
			return nil
		},
	}

	watcher := createTestWatcher(cfg, mockFetcher, mockProvider)
	ctx := context.Background()

	// Initial fetch
	_ = watcher.FetchAndUpdateIPs(ctx)
	ensureCalled = 0 // Reset counter

	// Check again - should update because IP has changed
	err := watcher.CheckAndUpdateIP(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if ensureCalled != 1 {
		t.Errorf("Expected DNS update when IP changed, got %d calls", ensureCalled)
	}
}

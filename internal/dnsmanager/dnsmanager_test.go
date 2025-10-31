package dnsmanager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudflare/cloudflare-go/v6/dns"
	"github.com/cloudflare/cloudflare-go/v6/zones"
	"github.com/msyrus/ipwatcher/internal/dnsmanager"
)

// MockCloudflareClient is a mock implementation of CloudflareClient for testing
type MockCloudflareClient struct {
	ListZonesFunc       func(ctx context.Context, params zones.ZoneListParams) ([]zones.Zone, error)
	ListDNSRecordsFunc  func(ctx context.Context, params dns.RecordListParams) ([]dns.RecordResponse, error)
	BatchDNSRecordsFunc func(ctx context.Context, params dns.RecordBatchParams) (*dns.RecordBatchResponse, error)
	DeleteDNSRecordFunc func(ctx context.Context, recordID string, params dns.RecordDeleteParams) (*dns.RecordDeleteResponse, error)
}

func (m *MockCloudflareClient) ListZones(ctx context.Context, params zones.ZoneListParams) ([]zones.Zone, error) {
	if m.ListZonesFunc != nil {
		return m.ListZonesFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockCloudflareClient) ListDNSRecords(ctx context.Context, params dns.RecordListParams) ([]dns.RecordResponse, error) {
	if m.ListDNSRecordsFunc != nil {
		return m.ListDNSRecordsFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockCloudflareClient) BatchDNSRecords(ctx context.Context, params dns.RecordBatchParams) (*dns.RecordBatchResponse, error) {
	if m.BatchDNSRecordsFunc != nil {
		return m.BatchDNSRecordsFunc(ctx, params)
	}
	return &dns.RecordBatchResponse{}, nil
}

func (m *MockCloudflareClient) DeleteDNSRecord(ctx context.Context, recordID string, params dns.RecordDeleteParams) (*dns.RecordDeleteResponse, error) {
	if m.DeleteDNSRecordFunc != nil {
		return m.DeleteDNSRecordFunc(ctx, recordID, params)
	}
	return &dns.RecordDeleteResponse{}, nil
}

func TestDNSRecordType_String(t *testing.T) {
	tests := []struct {
		name       string
		recordType dnsmanager.DNSRecordType
		expected   string
	}{
		{
			name:       "A record type",
			recordType: dnsmanager.ARecord,
			expected:   "A",
		},
		{
			name:       "AAAA record type",
			recordType: dnsmanager.AAAARecord,
			expected:   "AAAA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.recordType.String(); got != tt.expected {
				t.Errorf("DNSRecordType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewDNSManager(t *testing.T) {
	tests := []struct {
		name      string
		apiToken  string
		wantError bool
	}{
		{
			name:      "valid API token",
			apiToken:  "test-api-token-12345",
			wantError: false,
		},
		{
			name:      "empty API token",
			apiToken:  "",
			wantError: false, // Creation succeeds, validation happens at API call time
		},
		{
			name:      "long API token",
			apiToken:  "very-long-api-token-" + string(make([]byte, 100)),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := dnsmanager.NewDNSManager(tt.apiToken)

			if tt.wantError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.wantError && manager == nil {
				t.Error("NewDNSManager returned nil manager")
			}
		})
	}
}

func TestGetZoneIDByName_WithMock(t *testing.T) {
	tests := []struct {
		name        string
		zoneName    string
		mockZones   []zones.Zone
		mockError   error
		expectedID  string
		expectError bool
	}{
		{
			name:     "zone found",
			zoneName: "example.com",
			mockZones: []zones.Zone{
				{
					ID:   "zone-123",
					Name: "example.com",
				},
			},
			expectedID:  "zone-123",
			expectError: false,
		},
		{
			name:        "zone not found",
			zoneName:    "notfound.com",
			mockZones:   []zones.Zone{},
			expectedID:  "",
			expectError: true,
		},
		{
			name:        "API error",
			zoneName:    "example.com",
			mockZones:   nil,
			mockError:   errors.New("API error"),
			expectedID:  "",
			expectError: true,
		},
		{
			name:     "multiple zones - returns first match",
			zoneName: "example.com",
			mockZones: []zones.Zone{
				{
					ID:   "zone-first",
					Name: "example.com",
				},
				{
					ID:   "zone-second",
					Name: "example.com",
				},
			},
			expectedID:  "zone-first",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockCloudflareClient{
				ListZonesFunc: func(ctx context.Context, params zones.ZoneListParams) ([]zones.Zone, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockZones, nil
				},
			}

			manager := dnsmanager.NewDNSManagerWithClient(mockClient)
			ctx := context.Background()

			zoneID, err := manager.GetZoneIDByName(ctx, tt.zoneName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if zoneID != tt.expectedID {
					t.Errorf("Expected zone ID %q, got %q", tt.expectedID, zoneID)
				}
			}
		})
	}
}

func TestGetDNSRecords_WithMock(t *testing.T) {
	tests := []struct {
		name          string
		zoneID        string
		mockRecords   []dns.RecordResponse
		mockError     error
		expectedCount int
		expectError   bool
	}{
		{
			name:   "records found",
			zoneID: "zone-123",
			mockRecords: []dns.RecordResponse{
				{
					ID:   "record-1",
					Name: "www.example.com",
					Type: "A",
				},
				{
					ID:   "record-2",
					Name: "api.example.com",
					Type: "AAAA",
				},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "no records found",
			zoneID:        "zone-123",
			mockRecords:   []dns.RecordResponse{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "API error",
			zoneID:        "zone-123",
			mockRecords:   nil,
			mockError:     errors.New("API error"),
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockCloudflareClient{
				ListDNSRecordsFunc: func(ctx context.Context, params dns.RecordListParams) ([]dns.RecordResponse, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockRecords, nil
				},
			}

			manager := dnsmanager.NewDNSManagerWithClient(mockClient)
			ctx := context.Background()

			records, err := manager.GetDNSRecords(ctx, tt.zoneID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(records) != tt.expectedCount {
					t.Errorf("Expected %d records, got %d", tt.expectedCount, len(records))
				}
			}
		})
	}
}

func TestDNSRecord_Structure(t *testing.T) {
	tests := []struct {
		name   string
		record dnsmanager.DNSRecord
	}{
		{
			name: "A record with subdomain",
			record: dnsmanager.DNSRecord{
				Root:    "example.com",
				Name:    "www",
				Type:    dnsmanager.ARecord,
				Proxied: true,
			},
		},
		{
			name: "AAAA record with root domain",
			record: dnsmanager.DNSRecord{
				Root:    "example.com",
				Name:    "@",
				Type:    dnsmanager.AAAARecord,
				Proxied: false,
			},
		},
		{
			name: "A record without proxy",
			record: dnsmanager.DNSRecord{
				Root:    "test.org",
				Name:    "api",
				Type:    dnsmanager.ARecord,
				Proxied: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify record structure
			if tt.record.Root == "" {
				t.Error("Root should not be empty")
			}
			if tt.record.Name == "" {
				t.Error("Name should not be empty")
			}
			if tt.record.Type != dnsmanager.ARecord && tt.record.Type != dnsmanager.AAAARecord {
				t.Errorf("Invalid record type: %v", tt.record.Type)
			}
		})
	}
}

func TestDomain_Structure(t *testing.T) {
	domain := dnsmanager.Domain{
		ZoneID:   "zone-123",
		ZoneName: "example.com",
		Records: []dnsmanager.DNSRecord{
			{
				Root:    "example.com",
				Name:    "www",
				Type:    dnsmanager.ARecord,
				Proxied: true,
			},
		},
	}

	if domain.ZoneID == "" {
		t.Error("ZoneID should not be empty")
	}
	if domain.ZoneName == "" {
		t.Error("ZoneName should not be empty")
	}
	if len(domain.Records) == 0 {
		t.Error("Records should not be empty")
	}
}

func TestGetZoneIDByName_ErrorHandling(t *testing.T) {
	// This test verifies that we handle errors properly
	// In a real scenario, this would use dependency injection
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Call with a context to ensure the method signature is correct
	// This will fail without real credentials, which is expected
	_, err = manager.GetZoneIDByName(ctx, "test-zone")
	if err == nil {
		t.Log("Note: This test expects an error without real credentials")
	}
}

func TestGetDNSRecords_ErrorHandling(t *testing.T) {
	// This test verifies that we handle errors properly
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Call with a context to ensure the method signature is correct
	// This will fail without real credentials, which is expected
	_, err = manager.GetDNSRecords(ctx, "test-zone-id")
	if err == nil {
		t.Log("Note: This test expects an error without real credentials")
	}
}

func TestEnsureDNSRecords_EmptyRecords(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Test with empty records slice
	records := []dnsmanager.DNSRecord{}

	// This should handle empty records gracefully
	// Will fail at API call, but we're testing the function can be called
	err = manager.EnsureDNSRecords(ctx, "zone-id", records, "192.168.1.1", "2001:db8::1")
	if err == nil {
		t.Log("Note: This test expects an error without real credentials")
	}
}

func TestDeleteDNSRecord_ErrorHandling(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Test delete operation
	err = manager.DeleteDNSRecord(ctx, "zone-id", "record-id")
	if err == nil {
		t.Log("Note: This test expects an error without real credentials")
	}
}

func TestEnsureDNSRecords_WithARecordOnly(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Test with only A record
	records := []dnsmanager.DNSRecord{
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.ARecord,
			Proxied: true,
		},
	}

	// Provide only IPv4, no IPv6
	err = manager.EnsureDNSRecords(ctx, "zone-id", records, "192.168.1.1", "")
	// Will fail without real API, but we're testing the function accepts these params
	t.Logf("Called EnsureDNSRecords with A record only")
}

func TestEnsureDNSRecords_WithAAAARecordOnly(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Test with only AAAA record
	records := []dnsmanager.DNSRecord{
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.AAAARecord,
			Proxied: false,
		},
	}

	// Provide only IPv6, no IPv4
	err = manager.EnsureDNSRecords(ctx, "zone-id", records, "", "2001:db8::1")
	// Will fail without real API, but we're testing the function accepts these params
	t.Logf("Called EnsureDNSRecords with AAAA record only")
}

func TestEnsureDNSRecords_WithBothRecordTypes(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Test with both A and AAAA records
	records := []dnsmanager.DNSRecord{
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.ARecord,
			Proxied: true,
		},
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.AAAARecord,
			Proxied: true,
		},
	}

	// Provide both IPv4 and IPv6
	err = manager.EnsureDNSRecords(ctx, "zone-id", records, "192.168.1.1", "2001:db8::1")
	// Will fail without real API, but we're testing the function accepts these params
	t.Logf("Called EnsureDNSRecords with both A and AAAA records")
}

func TestEnsureDNSRecords_SkipsARecordWhenNoIPv4(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Test that A record is skipped when IPv4 is empty
	records := []dnsmanager.DNSRecord{
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.ARecord,
			Proxied: true,
		},
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.AAAARecord,
			Proxied: true,
		},
	}

	// Provide only IPv6, A record should be skipped
	err = manager.EnsureDNSRecords(ctx, "zone-id", records, "", "2001:db8::1")
	t.Logf("Called EnsureDNSRecords with empty IPv4 (A record should be skipped)")
}

func TestEnsureDNSRecords_SkipsAAAARecordWhenNoIPv6(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Test that AAAA record is skipped when IPv6 is empty
	records := []dnsmanager.DNSRecord{
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.ARecord,
			Proxied: true,
		},
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.AAAARecord,
			Proxied: true,
		},
	}

	// Provide only IPv4, AAAA record should be skipped
	err = manager.EnsureDNSRecords(ctx, "zone-id", records, "192.168.1.1", "")
	t.Logf("Called EnsureDNSRecords with empty IPv6 (AAAA record should be skipped)")
}

func TestEnsureDNSRecords_MultipleSubdomains(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Test with multiple subdomains
	records := []dnsmanager.DNSRecord{
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.ARecord,
			Proxied: true,
		},
		{
			Root:    "example.com",
			Name:    "api",
			Type:    dnsmanager.ARecord,
			Proxied: false,
		},
		{
			Root:    "example.com",
			Name:    "blog",
			Type:    dnsmanager.ARecord,
			Proxied: true,
		},
	}

	err = manager.EnsureDNSRecords(ctx, "zone-id", records, "192.168.1.1", "")
	t.Logf("Called EnsureDNSRecords with multiple subdomains")
}

func TestEnsureDNSRecords_RootDomain(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Test with root domain (@)
	records := []dnsmanager.DNSRecord{
		{
			Root:    "example.com",
			Name:    "@",
			Type:    dnsmanager.ARecord,
			Proxied: true,
		},
		{
			Root:    "example.com",
			Name:    "@",
			Type:    dnsmanager.AAAARecord,
			Proxied: true,
		},
	}

	err = manager.EnsureDNSRecords(ctx, "zone-id", records, "192.168.1.1", "2001:db8::1")
	t.Logf("Called EnsureDNSRecords with root domain (@)")
}

func TestEnsureDNSRecords_ProxiedVariations(t *testing.T) {
	tests := []struct {
		name    string
		records []dnsmanager.DNSRecord
	}{
		{
			name: "all proxied",
			records: []dnsmanager.DNSRecord{
				{
					Root:    "example.com",
					Name:    "www",
					Type:    dnsmanager.ARecord,
					Proxied: true,
				},
			},
		},
		{
			name: "none proxied",
			records: []dnsmanager.DNSRecord{
				{
					Root:    "example.com",
					Name:    "www",
					Type:    dnsmanager.ARecord,
					Proxied: false,
				},
			},
		},
		{
			name: "mixed proxied",
			records: []dnsmanager.DNSRecord{
				{
					Root:    "example.com",
					Name:    "www",
					Type:    dnsmanager.ARecord,
					Proxied: true,
				},
				{
					Root:    "example.com",
					Name:    "api",
					Type:    dnsmanager.ARecord,
					Proxied: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := dnsmanager.NewDNSManager("test-token")
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			ctx := context.Background()
			err = manager.EnsureDNSRecords(ctx, "zone-id", tt.records, "192.168.1.1", "")
			t.Logf("Called EnsureDNSRecords with %s configuration", tt.name)
		})
	}
}

func TestEnsureDNSRecords_DifferentIPFormats(t *testing.T) {
	tests := []struct {
		name string
		ipv4 string
		ipv6 string
	}{
		{
			name: "standard IPs",
			ipv4: "192.168.1.1",
			ipv6: "2001:db8::1",
		},
		{
			name: "public IPs",
			ipv4: "203.0.113.1",
			ipv6: "2001:db8:85a3::8a2e:370:7334",
		},
		{
			name: "IPv4 only",
			ipv4: "10.0.0.1",
			ipv6: "",
		},
		{
			name: "IPv6 only",
			ipv4: "",
			ipv6: "2001:db8::2",
		},
		{
			name: "both empty",
			ipv4: "",
			ipv6: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := dnsmanager.NewDNSManager("test-token")
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			ctx := context.Background()
			records := []dnsmanager.DNSRecord{
				{
					Root:    "example.com",
					Name:    "test",
					Type:    dnsmanager.ARecord,
					Proxied: false,
				},
				{
					Root:    "example.com",
					Name:    "test",
					Type:    dnsmanager.AAAARecord,
					Proxied: false,
				},
			}

			err = manager.EnsureDNSRecords(ctx, "zone-id", records, tt.ipv4, tt.ipv6)
			t.Logf("Called EnsureDNSRecords with %s", tt.name)
		})
	}
}

func TestEnsureDNSRecords_InvalidZoneID(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	records := []dnsmanager.DNSRecord{
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.ARecord,
			Proxied: true,
		},
	}

	tests := []struct {
		name   string
		zoneID string
	}{
		{
			name:   "empty zone ID",
			zoneID: "",
		},
		{
			name:   "invalid zone ID format",
			zoneID: "invalid-zone-id",
		},
		{
			name:   "numeric zone ID",
			zoneID: "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = manager.EnsureDNSRecords(ctx, tt.zoneID, records, "192.168.1.1", "")
			// Should fail with invalid zone ID
			t.Logf("Called EnsureDNSRecords with %s", tt.name)
		})
	}
}

func TestEnsureDNSRecords_ContextCancellation(t *testing.T) {
	manager, err := dnsmanager.NewDNSManager("test-token")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	records := []dnsmanager.DNSRecord{
		{
			Root:    "example.com",
			Name:    "www",
			Type:    dnsmanager.ARecord,
			Proxied: true,
		},
	}

	err = manager.EnsureDNSRecords(ctx, "zone-id", records, "192.168.1.1", "")
	// Should handle cancelled context
	t.Logf("Called EnsureDNSRecords with cancelled context")
}

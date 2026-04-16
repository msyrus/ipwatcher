package dnsmanager_test

import (
	"context"
	"testing"

	"github.com/cloudflare/cloudflare-go/v6/dns"
	"github.com/msyrus/ipwatcher/internal/dnsmanager"
)

func TestCloudflareEnsureDNSRecords_NoChangesSkipsBatch(t *testing.T) {
	batchCalls := 0

	mockClient := &MockCloudflareClient{
		ListDNSRecordsFunc: func(ctx context.Context, params dns.RecordListParams) ([]dns.RecordResponse, error) {
			return []dns.RecordResponse{{
				ID:      "rec-1",
				Name:    "www.example.com",
				Type:    dns.RecordResponseTypeA,
				Content: "203.0.113.10",
				Proxied: true,
			}}, nil
		},
		BatchDNSRecordsFunc: func(ctx context.Context, params dns.RecordBatchParams) (*dns.RecordBatchResponse, error) {
			batchCalls++
			return &dns.RecordBatchResponse{}, nil
		},
	}

	provider := dnsmanager.NewCloudflareProviderWithClient(mockClient)
	err := provider.EnsureDNSRecords(context.Background(), "zone-1", []dnsmanager.DNSRecord{{
		Root:    "example.com",
		Name:    "www",
		Type:    dnsmanager.ARecord,
		Proxied: true,
	}}, "203.0.113.10", "")
	if err != nil {
		t.Fatalf("EnsureDNSRecords returned error: %v", err)
	}

	if batchCalls != 0 {
		t.Fatalf("expected no batch calls for unchanged records, got %d", batchCalls)
	}
}

func TestCloudflareEnsureDNSRecords_ProxiedChangeTriggersBatch(t *testing.T) {
	batchCalls := 0

	mockClient := &MockCloudflareClient{
		ListDNSRecordsFunc: func(ctx context.Context, params dns.RecordListParams) ([]dns.RecordResponse, error) {
			return []dns.RecordResponse{{
				ID:      "rec-1",
				Name:    "www.example.com",
				Type:    dns.RecordResponseTypeA,
				Content: "203.0.113.10",
				Proxied: false,
			}}, nil
		},
		BatchDNSRecordsFunc: func(ctx context.Context, params dns.RecordBatchParams) (*dns.RecordBatchResponse, error) {
			batchCalls++
			return &dns.RecordBatchResponse{}, nil
		},
	}

	provider := dnsmanager.NewCloudflareProviderWithClient(mockClient)
	err := provider.EnsureDNSRecords(context.Background(), "zone-1", []dnsmanager.DNSRecord{{
		Root:    "example.com",
		Name:    "www",
		Type:    dnsmanager.ARecord,
		Proxied: true,
	}}, "203.0.113.10", "")
	if err != nil {
		t.Fatalf("EnsureDNSRecords returned error: %v", err)
	}

	if batchCalls != 1 {
		t.Fatalf("expected one batch call when proxied status changes, got %d", batchCalls)
	}
}

func TestCloudflareEnsureDNSRecords_EmptyIPsSkipAll(t *testing.T) {
	batchCalls := 0

	mockClient := &MockCloudflareClient{
		ListDNSRecordsFunc: func(ctx context.Context, params dns.RecordListParams) ([]dns.RecordResponse, error) {
			return []dns.RecordResponse{}, nil
		},
		BatchDNSRecordsFunc: func(ctx context.Context, params dns.RecordBatchParams) (*dns.RecordBatchResponse, error) {
			batchCalls++
			return &dns.RecordBatchResponse{}, nil
		},
	}

	provider := dnsmanager.NewCloudflareProviderWithClient(mockClient)
	err := provider.EnsureDNSRecords(context.Background(), "zone-1", []dnsmanager.DNSRecord{
		{Root: "example.com", Name: "www", Type: dnsmanager.ARecord, Proxied: true},
		{Root: "example.com", Name: "www", Type: dnsmanager.AAAARecord, Proxied: true},
	}, "", "")
	if err != nil {
		t.Fatalf("EnsureDNSRecords returned error: %v", err)
	}

	if batchCalls != 0 {
		t.Fatalf("expected zero batch calls when both IPv4/IPv6 are empty, got %d", batchCalls)
	}
}

package dnsmanager

import (
	"context"
)

// DNSProvider defines the interface for DNS operations across different providers
type DNSProvider interface {
	GetZoneIDByName(ctx context.Context, zoneName string) (string, error)
	EnsureDNSRecords(ctx context.Context, zoneID string, records []DNSRecord, ipv4, ipv6 string) error
}

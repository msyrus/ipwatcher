package dnsmanager

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/dns"
	"github.com/cloudflare/cloudflare-go/v6/option"
	"github.com/cloudflare/cloudflare-go/v6/zones"
)

// CloudflareClient defines the interface for Cloudflare operations
// This allows for dependency injection and mocking in tests
type CloudflareClient interface {
	ListZones(ctx context.Context, params zones.ZoneListParams) ([]zones.Zone, error)
	ListDNSRecords(ctx context.Context, params dns.RecordListParams) ([]dns.RecordResponse, error)
	BatchDNSRecords(ctx context.Context, params dns.RecordBatchParams) (*dns.RecordBatchResponse, error)
	DeleteDNSRecord(ctx context.Context, recordID string, params dns.RecordDeleteParams) (*dns.RecordDeleteResponse, error)
}

// RealCloudflareClient wraps the actual Cloudflare client
type RealCloudflareClient struct {
	client *cloudflare.Client
}

// NewRealCloudflareClient creates a new real Cloudflare client wrapper
func NewRealCloudflareClient(apiToken string) *RealCloudflareClient {
	client := cloudflare.NewClient(option.WithAPIToken(apiToken))
	return &RealCloudflareClient{client: client}
}

// ListZones implements CloudflareClient
func (r *RealCloudflareClient) ListZones(ctx context.Context, params zones.ZoneListParams) ([]zones.Zone, error) {
	page, err := r.client.Zones.List(ctx, params)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return []zones.Zone{}, nil
	}
	return page.Result, nil
}

// ListDNSRecords implements CloudflareClient
func (r *RealCloudflareClient) ListDNSRecords(ctx context.Context, params dns.RecordListParams) ([]dns.RecordResponse, error) {
	cur := r.client.DNS.Records.ListAutoPaging(ctx, params)
	records := []dns.RecordResponse{}
	for cur.Next() {
		if rec := cur.Current(); rec.Type == dns.RecordResponseTypeA || rec.Type == dns.RecordResponseTypeAAAA {
			records = append(records, rec)
		}
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// BatchDNSRecords implements CloudflareClient
func (r *RealCloudflareClient) BatchDNSRecords(ctx context.Context, params dns.RecordBatchParams) (*dns.RecordBatchResponse, error) {
	return r.client.DNS.Records.Batch(ctx, params)
}

// DeleteDNSRecord implements CloudflareClient
func (r *RealCloudflareClient) DeleteDNSRecord(ctx context.Context, recordID string, params dns.RecordDeleteParams) (*dns.RecordDeleteResponse, error) {
	return r.client.DNS.Records.Delete(ctx, recordID, params)
}

type DNSRecordType string

func (r DNSRecordType) String() string {
	return string(r)
}

const (
	ARecord    DNSRecordType = "A"
	AAAARecord DNSRecordType = "AAAA"
)

// DNSRecord represents a DNS record configuration
type DNSRecord struct {
	Root    string
	Name    string
	Type    DNSRecordType
	Proxied bool
}

// Domain represents a domain with its DNS records
type Domain struct {
	ZoneID   string
	ZoneName string
	Records  []DNSRecord
}

// DNSManager handles Cloudflare DNS operations
type DNSManager struct {
	client CloudflareClient
}

// NewDNSManager creates a new DNS manager instance
func NewDNSManager(apiToken string) (*DNSManager, error) {
	client := NewRealCloudflareClient(apiToken)
	return &DNSManager{
		client: client,
	}, nil
}

// NewDNSManagerWithClient creates a new DNS manager with a custom client (for testing)
func NewDNSManagerWithClient(client CloudflareClient) *DNSManager {
	return &DNSManager{
		client: client,
	}
}

// GetZoneIDByName retrieves the Zone ID for a given zone name
func (m *DNSManager) GetZoneIDByName(ctx context.Context, zoneName string) (string, error) {
	zones, err := m.client.ListZones(ctx, zones.ZoneListParams{Name: cloudflare.String(zoneName)})
	if err != nil {
		return "", fmt.Errorf("failed to list zones: %w", err)
	}
	if len(zones) == 0 {
		return "", fmt.Errorf("zone %s not found", zoneName)
	}
	return zones[0].ID, nil
}

// GetDNSRecords retrieves all DNS records for a domain
func (m *DNSManager) GetDNSRecords(ctx context.Context, zoneID string) ([]dns.RecordResponse, error) {
	records, err := m.client.ListDNSRecords(ctx, dns.RecordListParams{ZoneID: cloudflare.String(zoneID)})
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}
	return records, nil
}

type UpdateDNSRecord struct {
	ID string
	DNSRecord
}

func toDNSARecord(record DNSRecord, ipv4 string) dns.ARecordParam {
	return dns.ARecordParam{
		Name:    cloudflare.String(record.Name),
		Type:    cloudflare.F(dns.ARecordTypeA),
		Content: cloudflare.String(ipv4),
		Proxied: cloudflare.Bool(record.Proxied),
		TTL:     cloudflare.F(dns.TTL1), // Auto TTL
	}
}

func toDNSAAAARecord(record DNSRecord, ipv6 string) dns.AAAARecordParam {
	return dns.AAAARecordParam{
		Name:    cloudflare.String(record.Name),
		Type:    cloudflare.F(dns.AAAARecordTypeAAAA),
		Content: cloudflare.String(ipv6),
		Proxied: cloudflare.Bool(record.Proxied),
		TTL:     cloudflare.F(dns.TTL1), // Auto TTL
	}
}

func prepareBatchCreate(records []DNSRecord, ipv4, ipv6 string) []dns.RecordBatchParamsPostUnion {
	var newRecords []dns.RecordBatchParamsPostUnion
	for _, record := range records {
		switch record.Type {
		case ARecord:
			newRecords = append(newRecords, toDNSARecord(record, ipv4))
		case AAAARecord:
			newRecords = append(newRecords, toDNSAAAARecord(record, ipv6))
		}
	}

	return newRecords
}

func prepareBatchUpdate(records []UpdateDNSRecord, ipv4, ipv6 string) []dns.BatchPutUnionParam {
	var updateRecords []dns.BatchPutUnionParam
	for _, record := range records {
		switch record.Type {
		case ARecord:
			updateRecords = append(updateRecords, dns.BatchPutARecordParam{
				ID:           cloudflare.String(record.ID),
				ARecordParam: toDNSARecord(record.DNSRecord, ipv4),
			})
		case AAAARecord:
			updateRecords = append(updateRecords, dns.BatchPutAAAARecordParam{
				ID:              cloudflare.String(record.ID),
				AAAARecordParam: toDNSAAAARecord(record.DNSRecord, ipv6),
			})
		}
	}

	return updateRecords
}

func prepareRecordKey(record DNSRecord) string {
	name := record.Root
	if record.Name != "@" {
		name = record.Name + "." + record.Root
	}
	return name + "|" + record.Type.String()
}

// EnsureDNSRecords checks if the DNS records match the provided IPs and creates or updates them as necessary
func (m *DNSManager) EnsureDNSRecords(ctx context.Context, zoneID string, records []DNSRecord, ipv4, ipv6 string) error {
	existingRecords, err := m.GetDNSRecords(ctx, zoneID)
	if err != nil {
		return fmt.Errorf("failed to get existing DNS records: %w", err)
	}

	existingRecordMap := make(map[string]dns.RecordResponse)
	for _, rec := range existingRecords {
		if rec.Type == dns.RecordResponseTypeA || rec.Type == dns.RecordResponseTypeAAAA {
			existingRecordMap[rec.Name+"|"+string(rec.Type)] = rec
		}
	}
	var recordsToCreate []DNSRecord
	var recordsToUpdate []UpdateDNSRecord

	for _, record := range records {
		if record.Type == ARecord && ipv4 == "" {
			continue
		}
		if record.Type == AAAARecord && ipv6 == "" {
			continue
		}
		key := prepareRecordKey(record)
		existingRec, exists := existingRecordMap[key]
		if !exists {
			recordsToCreate = append(recordsToCreate, record)
			continue
		}

		var expectedContent string
		switch record.Type {
		case ARecord:
			expectedContent = ipv4
		case AAAARecord:
			expectedContent = ipv6
		}

		if existingRec.Content != expectedContent || existingRec.Proxied != record.Proxied {
			recordsToUpdate = append(recordsToUpdate, UpdateDNSRecord{
				ID:        existingRec.ID,
				DNSRecord: record,
			})
		}
	}

	if len(recordsToCreate) == 0 && len(recordsToUpdate) == 0 {
		log.Println("No DNS records to create or update")
		return nil
	}

	batchReq := dns.RecordBatchParams{
		ZoneID: cloudflare.String(zoneID),
	}

	if len(recordsToCreate) > 0 {
		batchReq.Posts = cloudflare.F(prepareBatchCreate(recordsToCreate, ipv4, ipv6))
	}

	if len(recordsToUpdate) > 0 {
		batchReq.Puts = cloudflare.F(prepareBatchUpdate(recordsToUpdate, ipv4, ipv6))
	}

	_, err = m.client.BatchDNSRecords(ctx, batchReq)
	if err != nil {
		return fmt.Errorf("failed to execute batch DNS record update: %w", err)
	}

	return nil
}

// DeleteDNSRecord deletes a DNS record by ID
func (m *DNSManager) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	_, err := m.client.DeleteDNSRecord(ctx, recordID, dns.RecordDeleteParams{
		ZoneID: cloudflare.String(zoneID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete DNS record %s: %w", recordID, err)
	}
	return nil
}

//go:build integration
// +build integration

package dnsmanager_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go/v6/dns"
	"github.com/msyrus/ipwatcher/internal/dnsmanager"
)

// Integration tests require:
// - CLOUDFLARE_API_TOKEN environment variable
// - CLOUDFLARE_TEST_ZONE_ID environment variable
// - CLOUDFLARE_TEST_ZONE_NAME environment variable (e.g., "example.com")
// Run with: go test -v -tags=integration ./internal/dnsmanager/

func skipIfNoCredentials(t *testing.T) (apiToken, zoneID, zoneName string) {
	apiToken = os.Getenv("CLOUDFLARE_API_TOKEN")
	zoneID = os.Getenv("CLOUDFLARE_TEST_ZONE_ID")
	zoneName = os.Getenv("CLOUDFLARE_TEST_ZONE_NAME")

	if apiToken == "" || zoneID == "" || zoneName == "" {
		t.Skip("Skipping integration test: CLOUDFLARE_API_TOKEN, CLOUDFLARE_TEST_ZONE_ID, and CLOUDFLARE_TEST_ZONE_NAME must be set")
	}

	return
}

func TestIntegration_GetZoneIDByName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiToken, expectedZoneID, zoneName := skipIfNoCredentials(t)

	manager, err := dnsmanager.NewDNSManager(apiToken)
	if err != nil {
		t.Fatalf("Failed to create DNS manager: %v", err)
	}

	ctx := context.Background()
	zoneID, err := manager.GetZoneIDByName(ctx, zoneName)
	if err != nil {
		t.Fatalf("GetZoneIDByName failed: %v", err)
	}

	if zoneID != expectedZoneID {
		t.Errorf("Expected zone ID %s, got %s", expectedZoneID, zoneID)
	}

	t.Logf("Successfully retrieved zone ID: %s", zoneID)
}

func TestIntegration_GetZoneIDByName_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiToken, _, _ := skipIfNoCredentials(t)

	manager, err := dnsmanager.NewDNSManager(apiToken)
	if err != nil {
		t.Fatalf("Failed to create DNS manager: %v", err)
	}

	ctx := context.Background()
	_, err = manager.GetZoneIDByName(ctx, "nonexistent-domain-12345.com")
	if err == nil {
		t.Fatal("Expected error for nonexistent zone, got nil")
	}

	t.Logf("Got expected error: %v", err)
}

func TestIntegration_GetDNSRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiToken, zoneID, _ := skipIfNoCredentials(t)

	manager, err := dnsmanager.NewDNSManager(apiToken)
	if err != nil {
		t.Fatalf("Failed to create DNS manager: %v", err)
	}

	ctx := context.Background()
	records, err := manager.GetDNSRecords(ctx, zoneID)
	if err != nil {
		t.Fatalf("GetDNSRecords failed: %v", err)
	}

	t.Logf("Retrieved %d DNS records", len(records))

	// Verify all records are A or AAAA type
	for _, rec := range records {
		if rec.Type != dns.RecordResponseTypeA && rec.Type != dns.RecordResponseTypeAAAA {
			t.Errorf("Unexpected record type: %s (expected A or AAAA)", rec.Type)
		}
	}
}

func TestIntegration_EnsureDNSRecords_CreateAndUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiToken, zoneID, zoneName := skipIfNoCredentials(t)

	manager, err := dnsmanager.NewDNSManager(apiToken)
	if err != nil {
		t.Fatalf("Failed to create DNS manager: %v", err)
	}

	ctx := context.Background()

	// Use a unique test subdomain to avoid conflicts
	testSubdomain := "ipwatcher-test-" + time.Now().Format("20060102-150405")
	testIPv4 := "203.0.113.100"
	testIPv6 := "2001:db8::100"

	records := []dnsmanager.DNSRecord{
		{
			Root:    zoneName,
			Name:    testSubdomain,
			Type:    dnsmanager.ARecord,
			Proxied: false,
		},
		{
			Root:    zoneName,
			Name:    testSubdomain,
			Type:    dnsmanager.AAAARecord,
			Proxied: false,
		},
	}

	// Step 1: Create the records
	t.Log("Creating DNS records...")
	err = manager.EnsureDNSRecords(ctx, zoneID, records, testIPv4, testIPv6)
	if err != nil {
		t.Fatalf("Failed to create DNS records: %v", err)
	}

	// Step 2: Verify they were created
	t.Log("Verifying records were created...")
	time.Sleep(2 * time.Second) // Give Cloudflare time to propagate

	allRecords, err := manager.GetDNSRecords(ctx, zoneID)
	if err != nil {
		t.Fatalf("Failed to get DNS records: %v", err)
	}

	fullName := testSubdomain + "." + zoneName
	var foundA, foundAAAA bool
	var recordIDs []string

	for _, rec := range allRecords {
		if rec.Name == fullName {
			recordIDs = append(recordIDs, rec.ID)
			if rec.Type == dns.RecordResponseTypeA {
				foundA = true
				if rec.Content != testIPv4 {
					t.Errorf("A record content mismatch: expected %s, got %s", testIPv4, rec.Content)
				}
			}
			if rec.Type == dns.RecordResponseTypeAAAA {
				foundAAAA = true
				if rec.Content != testIPv6 {
					t.Errorf("AAAA record content mismatch: expected %s, got %s", testIPv6, rec.Content)
				}
			}
		}
	}

	if !foundA {
		t.Error("A record was not created")
	}
	if !foundAAAA {
		t.Error("AAAA record was not created")
	}

	// Step 3: Update the records with new IPs
	t.Log("Updating DNS records with new IPs...")
	newIPv4 := "203.0.113.101"
	newIPv6 := "2001:db8::101"

	err = manager.EnsureDNSRecords(ctx, zoneID, records, newIPv4, newIPv6)
	if err != nil {
		t.Fatalf("Failed to update DNS records: %v", err)
	}

	// Step 4: Verify the updates
	t.Log("Verifying records were updated...")
	time.Sleep(2 * time.Second)

	allRecords, err = manager.GetDNSRecords(ctx, zoneID)
	if err != nil {
		t.Fatalf("Failed to get DNS records after update: %v", err)
	}

	foundA, foundAAAA = false, false
	for _, rec := range allRecords {
		if rec.Name == fullName {
			if rec.Type == dns.RecordResponseTypeA {
				foundA = true
				if rec.Content != newIPv4 {
					t.Errorf("A record not updated: expected %s, got %s", newIPv4, rec.Content)
				}
			}
			if rec.Type == dns.RecordResponseTypeAAAA {
				foundAAAA = true
				if rec.Content != newIPv6 {
					t.Errorf("AAAA record not updated: expected %s, got %s", newIPv6, rec.Content)
				}
			}
		}
	}

	if !foundA {
		t.Error("A record disappeared after update")
	}
	if !foundAAAA {
		t.Error("AAAA record disappeared after update")
	}

	// Step 5: Cleanup - delete the test records
	t.Log("Cleaning up test records...")
	for _, recordID := range recordIDs {
		err := manager.DeleteDNSRecord(ctx, zoneID, recordID)
		if err != nil {
			t.Logf("Warning: Failed to cleanup record %s: %v", recordID, err)
		}
	}

	t.Log("Integration test completed successfully")
}

func TestIntegration_EnsureDNSRecords_NoUpdatesNeeded(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiToken, zoneID, zoneName := skipIfNoCredentials(t)

	manager, err := dnsmanager.NewDNSManager(apiToken)
	if err != nil {
		t.Fatalf("Failed to create DNS manager: %v", err)
	}

	ctx := context.Background()

	testSubdomain := "ipwatcher-test-noupdate-" + time.Now().Format("20060102-150405")
	testIPv4 := "203.0.113.200"

	records := []dnsmanager.DNSRecord{
		{
			Root:    zoneName,
			Name:    testSubdomain,
			Type:    dnsmanager.ARecord,
			Proxied: false,
		},
	}

	// Create the record
	t.Log("Creating initial DNS record...")
	err = manager.EnsureDNSRecords(ctx, zoneID, records, testIPv4, "")
	if err != nil {
		t.Fatalf("Failed to create DNS record: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Call EnsureDNSRecords again with the same IP (should be a no-op)
	t.Log("Calling EnsureDNSRecords with same IP (should skip update)...")
	err = manager.EnsureDNSRecords(ctx, zoneID, records, testIPv4, "")
	if err != nil {
		t.Fatalf("Failed on second EnsureDNSRecords call: %v", err)
	}

	// Cleanup
	t.Log("Cleaning up...")
	allRecords, err := manager.GetDNSRecords(ctx, zoneID)
	if err == nil {
		fullName := testSubdomain + "." + zoneName
		for _, rec := range allRecords {
			if rec.Name == fullName {
				manager.DeleteDNSRecord(ctx, zoneID, rec.ID)
			}
		}
	}

	t.Log("No-update test completed successfully")
}

func TestIntegration_EnsureDNSRecords_ProxiedToggle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiToken, zoneID, zoneName := skipIfNoCredentials(t)

	manager, err := dnsmanager.NewDNSManager(apiToken)
	if err != nil {
		t.Fatalf("Failed to create DNS manager: %v", err)
	}

	ctx := context.Background()

	testSubdomain := "ipwatcher-test-proxy-" + time.Now().Format("20060102-150405")
	testIPv4 := "203.0.113.150"

	// Create with proxied=false
	records := []dnsmanager.DNSRecord{
		{
			Root:    zoneName,
			Name:    testSubdomain,
			Type:    dnsmanager.ARecord,
			Proxied: false,
		},
	}

	t.Log("Creating DNS record with proxied=false...")
	err = manager.EnsureDNSRecords(ctx, zoneID, records, testIPv4, "")
	if err != nil {
		t.Fatalf("Failed to create DNS record: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Update to proxied=true
	records[0].Proxied = true
	t.Log("Updating DNS record to proxied=true...")
	err = manager.EnsureDNSRecords(ctx, zoneID, records, testIPv4, "")
	if err != nil {
		t.Fatalf("Failed to update proxied status: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Verify the proxied status was updated
	allRecords, err := manager.GetDNSRecords(ctx, zoneID)
	if err != nil {
		t.Fatalf("Failed to get DNS records: %v", err)
	}

	fullName := testSubdomain + "." + zoneName
	var foundRecord dns.RecordResponse
	var recordID string

	for _, rec := range allRecords {
		if rec.Name == fullName && rec.Type == dns.RecordResponseTypeA {
			foundRecord = rec
			recordID = rec.ID
			break
		}
	}

	if recordID == "" {
		t.Fatal("Could not find created record")
	}

	if !foundRecord.Proxied {
		t.Error("Record proxied status was not updated to true")
	}

	// Cleanup
	t.Log("Cleaning up...")
	manager.DeleteDNSRecord(ctx, zoneID, recordID)

	t.Log("Proxied toggle test completed successfully")
}

func TestIntegration_EnsureDNSRecords_EmptyIPs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiToken, zoneID, zoneName := skipIfNoCredentials(t)

	manager, err := dnsmanager.NewDNSManager(apiToken)
	if err != nil {
		t.Fatalf("Failed to create DNS manager: %v", err)
	}

	ctx := context.Background()

	records := []dnsmanager.DNSRecord{
		{
			Root:    zoneName,
			Name:    "test-empty-ip",
			Type:    dnsmanager.ARecord,
			Proxied: false,
		},
		{
			Root:    zoneName,
			Name:    "test-empty-ip",
			Type:    dnsmanager.AAAARecord,
			Proxied: false,
		},
	}

	// Call with empty IPs - should skip both records
	t.Log("Calling EnsureDNSRecords with empty IPs...")
	err = manager.EnsureDNSRecords(ctx, zoneID, records, "", "")
	if err != nil {
		t.Fatalf("EnsureDNSRecords failed with empty IPs: %v", err)
	}

	t.Log("Empty IP test completed successfully (no records created)")
}

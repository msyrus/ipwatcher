//go:build integration
// +build integration

package dnsmanager_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/msyrus/ipwatcher/internal/dnsmanager"
)

// Route53 integration tests require:
// - ROUTE53_TEST_ZONE_NAME (e.g. "example.com")
// - AWS credentials available via the standard AWS SDK credential chain
// Run with: go test -v -p 1 -parallel 1 -tags=integration ./internal/dnsmanager/

func skipIfNoRoute53TestConfig(t *testing.T) string {
	t.Helper()

	zoneName := strings.TrimSpace(os.Getenv("ROUTE53_TEST_ZONE_NAME"))
	if zoneName == "" {
		t.Skip("Skipping Route53 integration test: ROUTE53_TEST_ZONE_NAME must be set")
	}

	return zoneName
}

func route53LoadClient(t *testing.T, ctx context.Context) *route53.Client {
	t.Helper()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Skipf("Skipping Route53 integration test: failed to load AWS config: %v", err)
	}

	return route53.NewFromConfig(cfg)
}

func route53FQDN(name, zoneName string) string {
	fqdn := zoneName
	if name != "@" {
		fqdn = name + "." + zoneName
	}
	if !strings.HasSuffix(fqdn, ".") {
		fqdn += "."
	}
	return fqdn
}

func route53FindRecordSet(ctx context.Context, client *route53.Client, zoneID, fqdn string, rrType types.RRType) (*types.ResourceRecordSet, error) {
	out, err := client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(zoneID),
		StartRecordName: aws.String(fqdn),
		StartRecordType: rrType,
		MaxItems:        aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}
	if len(out.ResourceRecordSets) == 0 {
		return nil, nil
	}

	rs := out.ResourceRecordSets[0]
	if aws.ToString(rs.Name) != fqdn || rs.Type != rrType {
		return nil, nil
	}
	return &rs, nil
}

func route53DeleteRecordIfExists(t *testing.T, ctx context.Context, client *route53.Client, zoneID, fqdn string, rrType types.RRType) {
	t.Helper()

	rs, err := route53FindRecordSet(ctx, client, zoneID, fqdn, rrType)
	if err != nil {
		t.Logf("cleanup lookup failed for %s %s: %v", rrType, fqdn, err)
		return
	}
	if rs == nil {
		return
	}

	_, err = client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action:            types.ChangeActionDelete,
				ResourceRecordSet: rs,
			}},
		},
	})
	if err != nil {
		t.Logf("cleanup delete failed for %s %s: %v", rrType, fqdn, err)
	}
}

func route53WaitForRecordValue(t *testing.T, ctx context.Context, client *route53.Client, zoneID, fqdn string, rrType types.RRType, expected string) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		rs, err := route53FindRecordSet(ctx, client, zoneID, fqdn, rrType)
		if err == nil && rs != nil && len(rs.ResourceRecords) == 1 && aws.ToString(rs.ResourceRecords[0].Value) == expected {
			return
		}
		time.Sleep(2 * time.Second)
	}

	rs, err := route53FindRecordSet(ctx, client, zoneID, fqdn, rrType)
	if err != nil {
		t.Fatalf("failed to verify %s %s: %v", rrType, fqdn, err)
	}
	if rs == nil {
		t.Fatalf("record not found for %s %s", rrType, fqdn)
	}
	if len(rs.ResourceRecords) != 1 {
		t.Fatalf("unexpected record values count for %s %s: %d", rrType, fqdn, len(rs.ResourceRecords))
	}
	got := aws.ToString(rs.ResourceRecords[0].Value)
	t.Fatalf("record value mismatch for %s %s: expected %s, got %s", rrType, fqdn, expected, got)
}

func TestIntegration_Route53_GetZoneIDByName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	zoneName := skipIfNoRoute53TestConfig(t)
	ctx := context.Background()

	provider, err := dnsmanager.NewRoute53Provider(ctx)
	if err != nil {
		t.Skipf("Skipping Route53 integration test: failed to create provider: %v", err)
	}

	zoneID, err := provider.GetZoneIDByName(ctx, zoneName)
	if err != nil {
		t.Fatalf("GetZoneIDByName failed: %v", err)
	}
	if zoneID == "" {
		t.Fatal("GetZoneIDByName returned empty zone ID")
	}

	t.Logf("Route53 hosted zone ID for %s: %s", zoneName, zoneID)
}

func TestIntegration_Route53_EnsureDNSRecords_CreateUpdateAndCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	zoneName := skipIfNoRoute53TestConfig(t)
	ctx := context.Background()

	provider, err := dnsmanager.NewRoute53Provider(ctx)
	if err != nil {
		t.Skipf("Skipping Route53 integration test: failed to create provider: %v", err)
	}

	zoneID, err := provider.GetZoneIDByName(ctx, zoneName)
	if err != nil {
		t.Fatalf("GetZoneIDByName failed: %v", err)
	}

	awsClient := route53LoadClient(t, ctx)

	testLabel := "ipwatcher-r53-test-" + time.Now().Format("20060102-150405")
	fqdn := route53FQDN(testLabel, zoneName)

	defer route53DeleteRecordIfExists(t, ctx, awsClient, zoneID, fqdn, types.RRTypeA)
	defer route53DeleteRecordIfExists(t, ctx, awsClient, zoneID, fqdn, types.RRTypeAaaa)

	records := []dnsmanager.DNSRecord{
		{Root: zoneName, Name: testLabel, Type: dnsmanager.ARecord},
		{Root: zoneName, Name: testLabel, Type: dnsmanager.AAAARecord},
	}

	ipv4First := "203.0.113.210"
	ipv6First := "2001:db8::210"

	if err := provider.EnsureDNSRecords(ctx, zoneID, records, ipv4First, ipv6First); err != nil {
		t.Fatalf("EnsureDNSRecords create failed: %v", err)
	}

	route53WaitForRecordValue(t, ctx, awsClient, zoneID, fqdn, types.RRTypeA, ipv4First)
	route53WaitForRecordValue(t, ctx, awsClient, zoneID, fqdn, types.RRTypeAaaa, ipv6First)

	ipv4Second := "203.0.113.211"
	ipv6Second := "2001:db8::211"

	if err := provider.EnsureDNSRecords(ctx, zoneID, records, ipv4Second, ipv6Second); err != nil {
		t.Fatalf("EnsureDNSRecords update failed: %v", err)
	}

	route53WaitForRecordValue(t, ctx, awsClient, zoneID, fqdn, types.RRTypeA, ipv4Second)
	route53WaitForRecordValue(t, ctx, awsClient, zoneID, fqdn, types.RRTypeAaaa, ipv6Second)
}

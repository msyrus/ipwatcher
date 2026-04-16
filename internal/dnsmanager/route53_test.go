package dnsmanager_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/msyrus/ipwatcher/internal/dnsmanager"
)

type mockRoute53Client struct {
	listHostedZonesByNameFunc    func(ctx context.Context, params *route53.ListHostedZonesByNameInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error)
	listResourceRecordSetsFunc   func(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
	changeResourceRecordSetsFunc func(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
}

func (m *mockRoute53Client) ListHostedZonesByName(ctx context.Context, params *route53.ListHostedZonesByNameInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error) {
	if m.listHostedZonesByNameFunc != nil {
		return m.listHostedZonesByNameFunc(ctx, params, optFns...)
	}
	return &route53.ListHostedZonesByNameOutput{}, nil
}

func (m *mockRoute53Client) ListResourceRecordSets(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
	if m.listResourceRecordSetsFunc != nil {
		return m.listResourceRecordSetsFunc(ctx, params, optFns...)
	}
	return &route53.ListResourceRecordSetsOutput{}, nil
}

func (m *mockRoute53Client) ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
	if m.changeResourceRecordSetsFunc != nil {
		return m.changeResourceRecordSetsFunc(ctx, params, optFns...)
	}
	return &route53.ChangeResourceRecordSetsOutput{}, nil
}

func TestRoute53GetZoneIDByName_StripsHostedZonePrefix(t *testing.T) {
	provider := dnsmanager.NewRoute53ProviderWithClient(&mockRoute53Client{
		listHostedZonesByNameFunc: func(ctx context.Context, params *route53.ListHostedZonesByNameInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error) {
			return &route53.ListHostedZonesByNameOutput{
				HostedZones: []types.HostedZone{{
					Name: aws.String("example.com."),
					Id:   aws.String("/hostedzone/Z123456"),
				}},
			}, nil
		},
	})

	zoneID, err := provider.GetZoneIDByName(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("GetZoneIDByName returned error: %v", err)
	}
	if zoneID != "Z123456" {
		t.Fatalf("expected Z123456, got %s", zoneID)
	}
}

func TestRoute53EnsureDNSRecords_PaginatesAndSkipsUnchanged(t *testing.T) {
	listCalls := 0
	changeCalls := 0

	provider := dnsmanager.NewRoute53ProviderWithClient(&mockRoute53Client{
		listResourceRecordSetsFunc: func(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
			listCalls++
			if listCalls == 1 {
				return &route53.ListResourceRecordSetsOutput{
					IsTruncated:    true,
					NextRecordName: aws.String("www.example.com."),
					NextRecordType: types.RRTypeA,
				}, nil
			}
			return &route53.ListResourceRecordSetsOutput{
				ResourceRecordSets: []types.ResourceRecordSet{{
					Name: aws.String("www.example.com."),
					Type: types.RRTypeA,
					ResourceRecords: []types.ResourceRecord{{
						Value: aws.String("203.0.113.10"),
					}},
				}},
			}, nil
		},
		changeResourceRecordSetsFunc: func(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
			changeCalls++
			return &route53.ChangeResourceRecordSetsOutput{}, nil
		},
	})

	err := provider.EnsureDNSRecords(context.Background(), "Z123", []dnsmanager.DNSRecord{{
		Root: "example.com",
		Name: "www",
		Type: dnsmanager.ARecord,
	}}, "203.0.113.10", "")
	if err != nil {
		t.Fatalf("EnsureDNSRecords returned error: %v", err)
	}

	if listCalls != 2 {
		t.Fatalf("expected 2 list calls due to pagination, got %d", listCalls)
	}
	if changeCalls != 0 {
		t.Fatalf("expected no change call for unchanged record, got %d", changeCalls)
	}
}

func TestRoute53EnsureDNSRecords_UpsertsWhenMissing(t *testing.T) {
	changeCalls := 0
	var captured *route53.ChangeResourceRecordSetsInput

	provider := dnsmanager.NewRoute53ProviderWithClient(&mockRoute53Client{
		listResourceRecordSetsFunc: func(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
			return &route53.ListResourceRecordSetsOutput{}, nil
		},
		changeResourceRecordSetsFunc: func(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
			changeCalls++
			captured = params
			return &route53.ChangeResourceRecordSetsOutput{}, nil
		},
	})

	err := provider.EnsureDNSRecords(context.Background(), "Z123", []dnsmanager.DNSRecord{{
		Root: "example.com",
		Name: "@",
		Type: dnsmanager.ARecord,
	}}, "203.0.113.20", "")
	if err != nil {
		t.Fatalf("EnsureDNSRecords returned error: %v", err)
	}

	if changeCalls != 1 {
		t.Fatalf("expected one change call, got %d", changeCalls)
	}
	if captured == nil || captured.ChangeBatch == nil || len(captured.ChangeBatch.Changes) != 1 {
		t.Fatalf("expected one upsert change")
	}

	change := captured.ChangeBatch.Changes[0]
	if change.ResourceRecordSet == nil || change.ResourceRecordSet.Name == nil {
		t.Fatalf("expected resource record set with name")
	}
	if got := aws.ToString(change.ResourceRecordSet.Name); got != "example.com." {
		t.Fatalf("expected apex fqdn example.com., got %s", got)
	}
}

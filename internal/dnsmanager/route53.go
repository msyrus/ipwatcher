package dnsmanager

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// Route53Client defines the subset of Route53 API methods used by the provider.
type Route53Client interface {
	ListHostedZonesByName(ctx context.Context, params *route53.ListHostedZonesByNameInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error)
	ListResourceRecordSets(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
	ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
}

// Route53Provider handles AWS Route53 DNS operations
type Route53Provider struct {
	client Route53Client
}

// NewRoute53Provider creates a new Route53 provider instance
func NewRoute53Provider(ctx context.Context) (*Route53Provider, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := route53.NewFromConfig(cfg)
	return &Route53Provider{
		client: client,
	}, nil
}

// NewRoute53ProviderWithClient creates a Route53 provider with a custom client (for testing).
func NewRoute53ProviderWithClient(client Route53Client) *Route53Provider {
	return &Route53Provider{client: client}
}

// GetZoneIDByName retrieves the Hosted Zone ID for a given zone name
func (p *Route53Provider) GetZoneIDByName(ctx context.Context, zoneName string) (string, error) {
	dotZoneName := zoneName
	if !strings.HasSuffix(dotZoneName, ".") {
		dotZoneName += "."
	}

	input := &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(dotZoneName),
	}

	output, err := p.client.ListHostedZonesByName(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to list hosted zones: %w", err)
	}

	for _, zone := range output.HostedZones {
		if *zone.Name == dotZoneName {
			// ID is in format /hostedzone/ID
			id := *zone.Id
			if after, ok := strings.CutPrefix(id, "/hostedzone/"); ok {
				id = after
			}
			return id, nil
		}
	}

	return "", fmt.Errorf("hosted zone %s not found", zoneName)
}

func (p *Route53Provider) listAllResourceRecordSets(ctx context.Context, zoneID string) ([]types.ResourceRecordSet, error) {
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	}

	var all []types.ResourceRecordSet
	for {
		output, err := p.client.ListResourceRecordSets(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list resource record sets: %w", err)
		}

		all = append(all, output.ResourceRecordSets...)

		if !output.IsTruncated {
			break
		}

		input.StartRecordName = output.NextRecordName
		input.StartRecordType = output.NextRecordType
		input.StartRecordIdentifier = output.NextRecordIdentifier
	}

	return all, nil
}

// EnsureDNSRecords checks if the DNS records match the provided IPs and updates them if necessary
func (p *Route53Provider) EnsureDNSRecords(ctx context.Context, zoneID string, records []DNSRecord, ipv4, ipv6 string) error {
	allRecords, err := p.listAllResourceRecordSets(ctx, zoneID)
	if err != nil {
		return err
	}

	existingRecordMap := make(map[string]types.ResourceRecordSet)
	for _, rs := range allRecords {
		if rs.Type == types.RRTypeA || rs.Type == types.RRTypeAaaa {
			existingRecordMap[*rs.Name+"|"+string(rs.Type)] = rs
		}
	}

	var changes []types.Change

	for _, record := range records {
		if record.Type == ARecord && ipv4 == "" {
			continue
		}
		if record.Type == AAAARecord && ipv6 == "" {
			continue
		}

		fqdn := record.Root
		if record.Name != "@" {
			fqdn = record.Name + "." + record.Root
		}
		if !strings.HasSuffix(fqdn, ".") {
			fqdn += "."
		}

		var targetIP string
		var rrType types.RRType
		if record.Type == ARecord {
			targetIP = ipv4
			rrType = types.RRTypeA
		} else {
			targetIP = ipv6
			rrType = types.RRTypeAaaa
		}

		key := fqdn + "|" + string(rrType)
		existing, exists := existingRecordMap[key]

		needsUpdate := !exists
		if exists {
			if len(existing.ResourceRecords) != 1 || *existing.ResourceRecords[0].Value != targetIP {
				needsUpdate = true
			}
		}

		if needsUpdate {
			changes = append(changes, types.Change{
				Action: types.ChangeActionUpsert,
				ResourceRecordSet: &types.ResourceRecordSet{
					Name: aws.String(fqdn),
					Type: rrType,
					TTL:  aws.Int64(300), // Default TTL
					ResourceRecords: []types.ResourceRecord{
						{
							Value: aws.String(targetIP),
						},
					},
				},
			})
		}
	}

	if len(changes) == 0 {
		log.Println("No Route53 DNS records to update")
		return nil
	}

	_, err = p.client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &types.ChangeBatch{
			Changes: changes,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to change resource record sets: %w", err)
	}

	log.Printf("Successfully updated %d records in Route53", len(changes))
	return nil
}

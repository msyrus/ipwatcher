package dnsmanager

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

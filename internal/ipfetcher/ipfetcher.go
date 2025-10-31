package ipfetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	ipv4URL = "https://api.ipify.org"
	ipv6URL = "https://api6.ipify.org"
	timeout = 10 * time.Second
)

// IPFetcher handles fetching public IP addresses
type IPFetcher struct {
	client *http.Client
}

// NewIPFetcher creates a new IP fetcher instance
func NewIPFetcher() *IPFetcher {
	return &IPFetcher{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetIPv4 fetches the public IPv4 address
func (f *IPFetcher) GetIPv4(ctx context.Context) (string, error) {
	return f.fetchIP(ctx, ipv4URL)
}

// GetIPv6 fetches the public IPv6 address
func (f *IPFetcher) GetIPv6(ctx context.Context) (string, error) {
	return f.fetchIP(ctx, ipv6URL)
}

// fetchIP performs the actual HTTP request to fetch IP
func (f *IPFetcher) fetchIP(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch IP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", fmt.Errorf("empty IP address received")
	}

	return ip, nil
}

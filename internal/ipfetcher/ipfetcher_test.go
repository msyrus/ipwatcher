package ipfetcher_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/msyrus/ipwatcher/internal/ipfetcher"
)

func TestNewIPFetcher(t *testing.T) {
	fetcher := ipfetcher.NewIPFetcher()
	if fetcher == nil {
		t.Fatal("NewIPFetcher returned nil")
	}
}

func TestGetIPv4_Success(t *testing.T) {
	// Create a test server that returns a mock IPv4 address
	expectedIP := "203.0.113.45"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedIP))
	}))
	defer server.Close()

	// Test the real GetIPv4 method (will use real API)
	// For proper testing, we'd need dependency injection or a way to override URLs
	fetcher := ipfetcher.NewIPFetcher()
	ctx := context.Background()

	// Note: This will make a real API call to ipify
	ip, err := fetcher.GetIPv4(ctx)

	if err != nil {
		t.Fatalf("GetIPv4 failed: %v", err)
	}

	// Just verify we got something that looks like an IP
	if ip == "" {
		t.Error("Expected non-empty IP address")
	}
}

func TestGetIPv6_Success(t *testing.T) {
	fetcher := ipfetcher.NewIPFetcher()
	ctx := context.Background()

	// Note: This will make a real API call to ipify
	// It may fail if the network doesn't support IPv6
	ip, err := fetcher.GetIPv6(ctx)

	if err != nil {
		// IPv6 might not be available in all environments, so we just log
		t.Logf("GetIPv6 failed (may be expected in IPv4-only environments): %v", err)
		return
	}

	if ip == "" {
		t.Error("Expected non-empty IP address")
	}
}

func TestGetIPv4_ContextCancellation(t *testing.T) {
	fetcher := ipfetcher.NewIPFetcher()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	_, err := fetcher.GetIPv4(ctx)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}
}

func TestGetIPv4_Timeout(t *testing.T) {
	fetcher := ipfetcher.NewIPFetcher()

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait a moment to ensure context expires
	time.Sleep(10 * time.Millisecond)

	_, err := fetcher.GetIPv4(ctx)

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
}

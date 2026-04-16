package ipfetcher_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/msyrus/ipwatcher/internal/ipfetcher"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestFetcher(fn roundTripFunc) *ipfetcher.IPFetcher {
	client := &http.Client{Transport: fn}
	return ipfetcher.NewIPFetcherWithClient(client)
}

func TestNewIPFetcher(t *testing.T) {
	fetcher := ipfetcher.NewIPFetcher()
	if fetcher == nil {
		t.Fatal("NewIPFetcher returned nil")
	}
}

func TestGetIPv4_Success(t *testing.T) {
	expectedIP := "203.0.113.45"
	fetcher := newTestFetcher(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api.ipify.org" {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(expectedIP)),
			Header:     make(http.Header),
		}, nil
	})
	ctx := context.Background()

	ip, err := fetcher.GetIPv4(ctx)
	if err != nil {
		t.Fatalf("GetIPv4 failed: %v", err)
	}
	if ip != expectedIP {
		t.Fatalf("expected %s, got %s", expectedIP, ip)
	}
}

func TestGetIPv6_Success(t *testing.T) {
	expectedIP := "2001:db8::1"
	fetcher := newTestFetcher(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api6.ipify.org" {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(expectedIP)),
			Header:     make(http.Header),
		}, nil
	})
	ctx := context.Background()

	ip, err := fetcher.GetIPv6(ctx)
	if err != nil {
		t.Fatalf("GetIPv6 failed: %v", err)
	}
	if ip != expectedIP {
		t.Fatalf("expected %s, got %s", expectedIP, ip)
	}
}

func TestGetIPv4_ContextCancellation(t *testing.T) {
	fetcher := newTestFetcher(func(req *http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fetcher.GetIPv4(ctx)
	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}
}

func TestGetIPv4_Timeout(t *testing.T) {
	fetcher := newTestFetcher(func(req *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	_, err := fetcher.GetIPv4(ctx)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
}

func TestGetIPv4_InvalidIPResponse(t *testing.T) {
	fetcher := newTestFetcher(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("not-an-ip")),
			Header:     make(http.Header),
		}, nil
	})

	_, err := fetcher.GetIPv4(context.Background())
	if err == nil {
		t.Fatal("Expected invalid IP error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid IP address") {
		t.Fatalf("expected invalid IP error, got: %v", err)
	}
}

func TestGetIPv4_Non200Response(t *testing.T) {
	fetcher := newTestFetcher(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader("unavailable")),
			Header:     make(http.Header),
		}, nil
	})

	_, err := fetcher.GetIPv4(context.Background())
	if err == nil {
		t.Fatal("Expected status code error, got nil")
	}
}

func TestGetIPv4_TransportError(t *testing.T) {
	fetcher := newTestFetcher(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network down")
	})

	_, err := fetcher.GetIPv4(context.Background())
	if err == nil {
		t.Fatal("Expected transport error, got nil")
	}
}

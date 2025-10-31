package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/msyrus/ipwatcher/internal/config"
	"github.com/msyrus/ipwatcher/internal/dnsmanager"
	"github.com/msyrus/ipwatcher/internal/ipfetcher"
)

// IPWatcher manages the IP monitoring and DNS update process
type IPWatcher struct {
	config        *config.Config
	ipFetcher     *ipfetcher.IPFetcher
	dnsManager    *dnsmanager.DNSManager
	zoneCache     *sync.Map // zone name -> zone ID cache
	currentIPv4   *atomic.Value
	currentIPv6   *atomic.Value
	refreshTicker *time.Ticker
	syncTicker    *time.Ticker
}

// NewIPWatcher creates a new IP watcher instance
func NewIPWatcher(cfg *config.Config, apiToken string) (*IPWatcher, error) {
	dnsManager, err := dnsmanager.NewDNSManager(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS manager: %w", err)
	}

	return &IPWatcher{
		config:      cfg,
		ipFetcher:   ipfetcher.NewIPFetcher(),
		dnsManager:  dnsManager,
		zoneCache:   &sync.Map{},
		currentIPv4: &atomic.Value{},
		currentIPv6: &atomic.Value{},
	}, nil
}

// Run starts the IP watcher daemon
func (w *IPWatcher) Run(ctx context.Context) error {
	log.Println("Starting IP Watcher daemon...")

	// Initial IP fetch
	if err := w.fetchAndUpdateIPs(ctx); err != nil {
		log.Printf("Warning: Initial IP fetch failed: %v", err)
	}

	// Create tickers for refresh and sync
	refreshInterval := time.Duration(float64(time.Second) / w.config.RefreshRate)
	syncInterval := time.Duration(float64(time.Minute) / w.config.SyncRate)

	w.refreshTicker = time.NewTicker(refreshInterval)
	defer w.refreshTicker.Stop()

	w.syncTicker = time.NewTicker(syncInterval)
	defer w.syncTicker.Stop()

	log.Printf("Refresh interval: %v (%.2f times per second)", refreshInterval, w.config.RefreshRate)
	log.Printf("Sync interval: %v (%.2f times per minute)", syncInterval, w.config.SyncRate)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down IP Watcher daemon...")
			return ctx.Err()

		case <-w.refreshTicker.C:
			if err := w.checkAndUpdateIP(ctx); err != nil {
				log.Printf("Error checking IP: %v", err)
			}

		case <-w.syncTicker.C:
			if err := w.verifyDNSRecords(ctx); err != nil {
				log.Printf("Error verifying DNS records: %v", err)
			}
		}
	}
}

// fetchAndUpdateIPs fetches current IPs and updates DNS if needed
func (w *IPWatcher) fetchAndUpdateIPs(ctx context.Context) error {
	// Fetch IPv4
	ipv4, err := w.ipFetcher.GetIPv4(ctx)
	if err != nil {
		log.Printf("Failed to fetch IPv4: %v", err)
	} else {
		w.currentIPv4.Store(ipv4)
		log.Printf("Current IPv4: %s", ipv4)
	}

	// Fetch IPv6
	if w.config.SupportsIPv6 {
		ipv6, err := w.ipFetcher.GetIPv6(ctx)
		if err != nil {
			log.Printf("Failed to fetch IPv6: %v", err)
		} else {
			w.currentIPv6.Store(ipv6)
			log.Printf("Current IPv6: %s", ipv6)
		}
	}

	// Update DNS records
	return w.updateAllDNSRecords(ctx)
}

// checkAndUpdateIP checks if IP has changed and updates DNS if needed
func (w *IPWatcher) checkAndUpdateIP(ctx context.Context) error {
	oldIPv4, _ := w.currentIPv4.Load().(string)
	oldIPv6, _ := w.currentIPv6.Load().(string)

	// Fetch current IPs
	newIPv4, err := w.ipFetcher.GetIPv4(ctx)
	if err != nil {
		log.Printf("Failed to fetch IPv4: %v", err)
	}

	newIPv6 := ""
	if w.config.SupportsIPv6 {
		newIPv6, err = w.ipFetcher.GetIPv6(ctx)
		if err != nil {
			// IPv6 might not be available, just log it
			log.Printf("Failed to fetch IPv6: %v", err)
		}
	}

	// Check if IPs have changed
	ipv4Changed := newIPv4 != oldIPv4 && newIPv4 != ""
	ipv6Changed := newIPv6 != oldIPv6 && newIPv6 != ""

	if ipv4Changed {
		log.Printf("IPv4 changed: %s -> %s", oldIPv4, newIPv4)
		w.currentIPv4.Store(newIPv4)
	}
	if ipv6Changed {
		log.Printf("IPv6 changed: %s -> %s", oldIPv6, newIPv6)
		w.currentIPv6.Store(newIPv6)
	}
	if ipv4Changed || ipv6Changed {
		w.syncTicker.Reset(time.Duration(float64(time.Minute) / w.config.SyncRate)) // Reset sync ticker on IP change

		return w.updateAllDNSRecords(ctx)
	}

	return nil
}

// getZoneID retrieves the zone ID for a domain, using cache if available
func (w *IPWatcher) getZoneID(ctx context.Context, zoneName string) (string, error) {
	zoneID, exists := w.zoneCache.Load(zoneName)

	if exists {
		return zoneID.(string), nil
	}

	// Fetch zone ID from Cloudflare
	zID, err := w.dnsManager.GetZoneIDByName(ctx, zoneName)
	if err != nil {
		return "", err
	}

	// Cache it
	w.zoneCache.Store(zoneName, zID)

	return zID, nil
}

// updateAllDNSRecords updates DNS records for all configured domains
func (w *IPWatcher) updateAllDNSRecords(ctx context.Context) error {
	ipv4, _ := w.currentIPv4.Load().(string)
	ipv6, _ := w.currentIPv6.Load().(string)

	var lastErr error
	for _, domain := range w.config.Domains {
		// Get zone ID
		zoneID, err := w.getZoneID(ctx, domain.ZoneName)
		if err != nil {
			log.Printf("Failed to get zone ID for %s: %v", domain.ZoneName, err)
			lastErr = err
			continue
		}

		// Convert config records to DNS manager records
		var dnsRecords []dnsmanager.DNSRecord
		for _, record := range domain.Records {
			dnsRecords = append(dnsRecords, dnsmanager.DNSRecord{
				Root:    domain.ZoneName,
				Name:    record.Name,
				Type:    dnsmanager.DNSRecordType(record.Type),
				Proxied: record.Proxied,
			})
		}

		// Use EnsureDNSRecords to batch create/update
		if err := w.dnsManager.EnsureDNSRecords(ctx, zoneID, dnsRecords, ipv4, ipv6); err != nil {
			log.Printf("Failed to ensure DNS records for %s: %v", domain.ZoneName, err)
			lastErr = err
		} else {
			log.Printf("DNS records for %s updated successfully", domain.ZoneName)
		}
	}

	return lastErr
}

// verifyDNSRecords verifies that all DNS records are up-to-date
func (w *IPWatcher) verifyDNSRecords(ctx context.Context) error {
	ipv4, _ := w.currentIPv4.Load().(string)
	ipv6, _ := w.currentIPv6.Load().(string)

	log.Println("Verifying DNS records...")

	var lastErr error
	for _, domain := range w.config.Domains {
		// Get zone ID
		zoneID, err := w.getZoneID(ctx, domain.ZoneName)
		if err != nil {
			log.Printf("Failed to get zone ID for %s: %v", domain.ZoneName, err)
			lastErr = err
			continue
		}

		// Convert config records to DNS manager records
		var dnsRecords []dnsmanager.DNSRecord
		for _, record := range domain.Records {
			dnsRecords = append(dnsRecords, dnsmanager.DNSRecord{
				Root:    domain.ZoneName,
				Name:    record.Name,
				Type:    dnsmanager.DNSRecordType(record.Type),
				Proxied: record.Proxied,
			})
		}

		// Use EnsureDNSRecords which will update only if needed
		if err := w.dnsManager.EnsureDNSRecords(ctx, zoneID, dnsRecords, ipv4, ipv6); err != nil {
			log.Printf("Failed to verify/update DNS records for %s: %v", domain.ZoneName, err)
			lastErr = err
		} else {
			log.Printf("DNS records for %s are up-to-date", domain.ZoneName)
		}
	}

	return lastErr
}

func main() {
	// Load configuration
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config.yaml"
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Get Cloudflare API token
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	if apiToken == "" {
		log.Fatal("CLOUDFLARE_API_TOKEN environment variable is required")
	}

	// Create IP watcher
	watcher, err := NewIPWatcher(cfg, apiToken)
	if err != nil {
		log.Fatalf("Failed to create IP watcher: %v", err)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Run the watcher
	if err := watcher.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("IP watcher error: %v", err)
	}

	log.Println("IP Watcher daemon stopped")
}

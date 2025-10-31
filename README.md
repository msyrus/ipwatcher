# ipwatcher

[![Build and Test](https://github.com/msyrus/ipwatcher/actions/workflows/build.yml/badge.svg)](https://github.com/msyrus/ipwatcher/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/msyrus/ipwatcher)](https://goreportcard.com/report/github.com/msyrus/ipwatcher)
[![codecov](https://codecov.io/gh/msyrus/ipwatcher/branch/main/graph/badge.svg)](https://codecov.io/gh/msyrus/ipwatcher)
[![Go Version](https://img.shields.io/github/go-mod/go-version/msyrus/ipwatcher)](https://github.com/msyrus/ipwatcher)
[![License](https://img.shields.io/github/license/msyrus/ipwatcher)](https://github.com/msyrus/ipwatcher/blob/main/LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/msyrus/ipwatcher)](https://hub.docker.com/r/msyrus/ipwatcher)

A daemon service that monitors your server's public IP address and automatically updates A and AAAA DNS records in Cloudflare for your configured domains and subdomains.

## Features

- **Automatic IP Detection**: Fetches both IPv4 and IPv6 addresses from ipify API
- **Cloudflare DNS Management**: Automatically updates A (IPv4) and AAAA (IPv6) records
- **Configurable Rates**:
  - **Refresh Rate**: How many times per second to check for IP changes
  - **Sync Rate**: How many times per minute to verify DNS records are up-to-date
- **Multi-Domain Support**: Manage multiple domains and subdomains from a single configuration
- **Systemd Integration**: Run as a proper system daemon with automatic restart
- **Docker Support**: Run in containers with Docker or Docker Compose
- **Graceful Shutdown**: Handles SIGTERM/SIGINT signals properly

## Prerequisites

- Go 1.21 or later (for building from source)
- OR Docker (for containerized deployment)
- A Cloudflare account with API token
- Linux system with systemd (for daemon mode, optional)

## Installation

### 1. Clone the repository

```bash
git clone https://github.com/msyrus/ipwatcher.git
cd ipwatcher
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Build the binary

```bash
go build -o ipwatcher ./cmd/ipwatcher
```

### 4. Configure the service

#### Create configuration file

Copy the example configuration:

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` with your settings:

```yaml
# How many times per second to check for IP changes (0.1 = once every 10 seconds)
refresh_rate: 0.1

# How many times per minute to verify DNS records (1 = once per minute)
sync_rate: 1

domains:
  - zone_name: "example.com"
    records:
      - name: "@"
        type: A
        proxied: false
      - name: "www"
        type: A
        proxied: true
```

#### Set up environment variables

Create a `.env` file:

```bash
echo "CLOUDFLARE_API_TOKEN=your_api_token_here" > .env
```

**Creating a Cloudflare API Token:**

1. Go to [https://dash.cloudflare.com/profile/api-tokens](https://dash.cloudflare.com/profile/api-tokens)
2. Click "Create Token"
3. Use the "Edit zone DNS" template or create a custom token with:
   - Permissions: `Zone` → `DNS` → `Edit`
   - Zone Resources: `Include` → `Specific zone` → Select your domain(s)
4. Copy the token and add it to `.env`

### 5. Install as systemd service (Optional)

Run the installation script:

```bash
chmod +x install.sh
sudo ./install.sh
```

This will:

- Create an `ipwatcher` system user
- Install the binary to `/opt/ipwatcher`
- Copy configuration files
- Install and configure the systemd service

### 6. Docker Deployment (Alternative)

See [DOCKER.md](DOCKER.md) for complete Docker deployment guide.

Quick start with Docker Compose:

```bash
# Create config and .env files
cp config.yaml.example config.yaml
echo "CLOUDFLARE_API_TOKEN=your_token" > .env

# Start with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f
```

## Usage

### Running with Docker Compose

```bash
# Start the service
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the service
docker-compose down
```

### Running manually

```bash
# Set environment variables
export CLOUDFLARE_API_TOKEN="your_token"
export CONFIG_FILE="config.yaml"

# Run the daemon
./ipwatcher
```

### Running as systemd service

```bash
# Enable service to start on boot
sudo systemctl enable ipwatcher

# Start the service
sudo systemctl start ipwatcher

# Check status
sudo systemctl status ipwatcher

# View logs
sudo journalctl -u ipwatcher -f

# Stop the service
sudo systemctl stop ipwatcher

# Restart the service
sudo systemctl restart ipwatcher
```

## Configuration Reference

### Global Settings

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `refresh_rate` | float | Times per second to check IP changes | `0.1` (every 10s) |
| `sync_rate` | float | Times per minute to verify DNS records | `1` (every minute) |

### Domain Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `zone_name` | string | Yes | Domain name (e.g., "example.com") |
| `records` | array | Yes | List of DNS records to manage |

**Note**: The zone ID is automatically looked up from the zone name using the Cloudflare API.

### Record Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Full domain name (e.g., "example.com" or "www.example.com") |
| `type` | string | Yes | Record type: `A` (IPv4) or `AAAA` (IPv6) |
| `proxied` | bool | Yes | Whether to proxy through Cloudflare CDN |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `CLOUDFLARE_API_TOKEN` | Yes | Cloudflare API token with DNS edit permissions |
| `CONFIG_FILE` | No | Path to configuration file (default: `config.yaml`) |

## How It Works

1. **IP Monitoring**: The daemon periodically queries the ipify API to get the current public IPv4 and IPv6 addresses
2. **Change Detection**: When an IP change is detected, it immediately updates all configured DNS records
3. **Periodic Verification**: At the configured sync rate, the daemon verifies all DNS records are correct and updates them if needed
4. **Graceful Updates**: Only updates records that have actually changed to minimize API calls

## Logging

The daemon logs all operations including:

- IP address changes
- DNS record updates
- Verification checks
- Errors and warnings

When running as a systemd service, logs are sent to journald and can be viewed with:

```bash
sudo journalctl -u ipwatcher -f
```

## Troubleshooting

### Service won't start

Check the logs:

```bash
sudo journalctl -u ipwatcher -n 50
```

Common issues:

- Missing or invalid Cloudflare API token
- Invalid configuration file
- Incorrect zone name or permissions

### DNS records not updating

1. Verify your API token has the correct permissions
2. Check that the zone name in config matches your actual domain
3. Ensure the record names are in the correct format (use "@" for root domain, or just the subdomain like "www")

### IPv6 not working

IPv6 might not be available on your network. The daemon will log warnings but continue operating with IPv4 only.

## Development

### Project Structure

```text
ipwatcher/
├── cmd/
│   └── ipwatcher/          # Main application entry point
│       └── main.go
├── internal/
│   ├── config/             # Configuration management
│   │   └── config.go
│   ├── dnsmanager/         # Cloudflare DNS operations
│   │   └── dnsmanager.go
│   └── ipfetcher/          # IP address fetching
│       └── ipfetcher.go
├── config.yaml.example     # Example configuration
├── ipwatcher.service       # Systemd service file
├── install.sh              # Installation script
├── go.mod                  # Go module definition
└── README.md
```

### Building

```bash
go build -o ipwatcher ./cmd/ipwatcher
```

### Testing

Run tests:

```bash
# All tests (unit + integration, requires credentials for integration)
make test

# Unit tests only (integration tests excluded)
make test-unit

# With coverage (includes integration tests if credentials available)
make test-coverage

# Unit tests coverage only
make test-coverage-unit

# Short tests only
make test-short

# Benchmarks
make bench
```

#### Integration Tests

Integration tests for the DNS manager package test against actual Cloudflare API. These tests use Go build tags and are excluded from normal test runs.

**Note**: Integration tests require Cloudflare credentials and use the `integration` build tag.

To run integration tests:

```bash
# Set required environment variables
export CLOUDFLARE_API_TOKEN="your-api-token"
export CLOUDFLARE_TEST_ZONE_ID="your-zone-id"
export CLOUDFLARE_TEST_ZONE_NAME="example.com"

# Run integration tests (using Makefile - recommended)
make test-integration

# Or run directly with build tags
go test -v -tags=integration ./internal/dnsmanager/
```

For detailed information about integration tests, see [internal/dnsmanager/INTEGRATION_TESTS.md](internal/dnsmanager/INTEGRATION_TESTS.md).

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

msyrus

## Acknowledgments

- [ipify](https://www.ipify.org/) for the IP detection API
- [Cloudflare](https://www.cloudflare.com/) for DNS management
- [cloudflare-go](https://github.com/cloudflare/cloudflare-go) for the Go API client

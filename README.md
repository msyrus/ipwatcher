# ipwatcher

[![Build and Test](https://github.com/msyrus/ipwatcher/actions/workflows/build.yml/badge.svg)](https://github.com/msyrus/ipwatcher/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/msyrus/ipwatcher)](https://goreportcard.com/report/github.com/msyrus/ipwatcher)
[![codecov](https://codecov.io/gh/msyrus/ipwatcher/branch/main/graph/badge.svg)](https://codecov.io/gh/msyrus/ipwatcher)
[![Go Version](https://img.shields.io/github/go-mod/go-version/msyrus/ipwatcher)](https://github.com/msyrus/ipwatcher)
[![License](https://img.shields.io/github/license/msyrus/ipwatcher)](https://github.com/msyrus/ipwatcher/blob/main/LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/msyrus/ipwatcher)](https://hub.docker.com/r/msyrus/ipwatcher)

`ipwatcher` monitors your public IP address and keeps DNS `A` and `AAAA` records in sync across Cloudflare and AWS Route 53.

It is designed for home labs, self-hosted services, VPN endpoints, and tiny-but-stubborn servers whose public IP likes to wander off unsupervised.

## Features

- Automatic public IPv4 detection, with optional IPv6 support
- Per-zone provider selection: `cloudflare` or `route53`
- Mixed-provider configs in a single deployment
- Immediate DNS updates on IP change plus scheduled reconciliation
- Cloudflare proxy support for `A` and `AAAA` records
- Route 53 hosted zone discovery by zone name
- Linux systemd service and Docker/Docker Compose support
- Graceful shutdown on `SIGINT` and `SIGTERM`

## Supported providers

### Cloudflare

- Uses `CLOUDFLARE_API_TOKEN`
- Supports proxied and non-proxied `A` / `AAAA` records
- Automatically looks up the zone ID from `zone_name`

### AWS Route 53

- Uses the standard AWS SDK credential chain
- Common setup is `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_REGION`
- Automatically looks up the hosted zone ID from `zone_name`
- Ignores the `proxied` setting because Route 53 does not have a Cloudflare-style proxy mode

## Prerequisites

- Go 1.21+ if building from source
- or Docker / Docker Compose for container deployment
- Cloudflare API access if you use Cloudflare-managed zones
- AWS credentials with Route 53 permissions if you use Route 53-managed zones
- Linux with systemd if you want to run it as a service

## Quick start

### 1. Clone and build

```bash
git clone https://github.com/msyrus/ipwatcher.git
cd ipwatcher
go mod download
go build -o ipwatcher ./cmd/ipwatcher
```

### 2. Create your config

```bash
cp config.yaml.example config.yaml
```

Minimal mixed-provider example:

```yaml
refresh_rate: 0.1
sync_rate: 1
supports_ipv6: true

domains:
  - zone_name: "example.com"
    provider: "cloudflare"
    records:
      - name: "@"
        type: A
        proxied: false
      - name: "www"
        type: A
        proxied: true
      - name: "www"
        type: AAAA
        proxied: true

  - zone_name: "example.net"
    provider: "route53"
    records:
      - name: "home"
        type: A
      - name: "vpn"
        type: AAAA
```

### 3. Add credentials

```bash
cp .env.example .env
```

Then edit `.env` and keep only the credentials relevant to the providers you use.

Example:

```bash
CLOUDFLARE_API_TOKEN=your_cloudflare_token
AWS_ACCESS_KEY_ID=your_aws_access_key_id
AWS_SECRET_ACCESS_KEY=your_aws_secret_access_key
AWS_REGION=us-east-1
```

### 4. Run it manually

```bash
export CONFIG_FILE=config.yaml
set -a
source .env
set +a
./ipwatcher
```

### 5. Or install it as a service

```bash
sudo ./install.sh
sudo systemctl enable --now ipwatcher
sudo journalctl -u ipwatcher -f
```

For a shorter guided setup, see [QUICKSTART.md](QUICKSTART.md).

## Docker and Docker Compose

```bash
cp config.yaml.example config.yaml
cp .env.example .env
mkdir -p logs
docker compose up -d
docker compose logs -f
```

If you are using Route 53 in Docker, fill in the AWS variables in `.env` before starting the container.

## Configuration reference

### Global settings

| Field | Type | Description | Example |
| ----- | ---- | ----------- | ------- |
| `refresh_rate` | float | How many times per second to check the public IP | `0.1` |
| `sync_rate` | float | How many times per minute to reconcile DNS records | `1` |
| `supports_ipv6` | bool | Enable IPv6 fetching and allow `AAAA` records | `false` |

`supports_ipv6` must be `true` if any configured record uses type `AAAA`.

### Domain settings

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `zone_name` | string | Yes | DNS zone / hosted zone name, such as `example.com` |
| `provider` | string | No | `cloudflare` or `route53`; defaults to `cloudflare` |
| `records` | array | Yes | Records to manage inside the zone |

### Record settings

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `name` | string | Yes | Relative record name: use `@` for the zone apex, or labels like `www`, `vpn`, `home` |
| `type` | string | Yes | `A` or `AAAA` |
| `proxied` | bool | No | Cloudflare-only proxy flag; ignored by Route 53 |

For `zone_name: "example.com"`:

- `name: "@"` manages `example.com`
- `name: "www"` manages `www.example.com`
- `name: "vpn"` manages `vpn.example.com`

## Environment variables

| Variable | Required | Description |
| -------- | -------- | ----------- |
| `CLOUDFLARE_API_TOKEN` | If using Cloudflare | Cloudflare API token with DNS edit permissions |
| `AWS_ACCESS_KEY_ID` | Usually, if using Route 53 | AWS access key for Route 53 |
| `AWS_SECRET_ACCESS_KEY` | Usually, if using Route 53 | AWS secret access key |
| `AWS_SESSION_TOKEN` | Optional | AWS session token for temporary credentials |
| `AWS_REGION` | Recommended for Route 53 | Region passed to the AWS SDK, commonly `us-east-1` |
| `CONFIG_FILE` | No | Config file path; defaults to `config.yaml` |

Route 53 authentication uses the AWS SDK default credential chain, so environment variables are the easiest option, not the only option.

## Provider-specific notes

### Cloudflare token permissions

Create a token with at least:

- `Zone` → `DNS` → `Edit`
- Zone scope for the domains you want to manage

### Route 53 IAM permissions

The Route 53 provider needs permission to:

- list hosted zones by name
- list resource record sets
- change resource record sets

Typical actions are:

- `route53:ListHostedZonesByName`
- `route53:ListResourceRecordSets`
- `route53:ChangeResourceRecordSets`

## How it works

1. Fetch the current public IPv4 address and, when enabled, the public IPv6 address
2. Cache the last known values in memory
3. Update managed DNS records whenever an IP changes
4. Periodically verify all configured records and reconcile drift

Only records that need to change are updated, which keeps API traffic tidy.

## Running as a systemd service

After installation:

```bash
sudo systemctl enable ipwatcher
sudo systemctl start ipwatcher
sudo systemctl status ipwatcher
sudo journalctl -u ipwatcher -f
```

The service reads environment variables from `/opt/ipwatcher/.env` and the configuration from `/opt/ipwatcher/config.yaml`.

## Troubleshooting

### Service fails to start

Check recent logs:

```bash
sudo journalctl -u ipwatcher -n 50
```

Common causes:

- missing Cloudflare or AWS credentials for the configured provider
- invalid YAML in `config.yaml`
- `AAAA` records configured while `supports_ipv6` is `false`
- `zone_name` does not match the Cloudflare zone or Route 53 hosted zone name

### Records are not updating

Check these first:

1. `zone_name` exactly matches the authoritative zone
2. `name` uses `@` or a relative label, not a full FQDN
3. the provider credentials have permission to read and update DNS
4. the host can reach the public internet to resolve its current IP

### IPv6 warnings

If IPv6 is enabled but not available on the host or network, the daemon logs the fetch failure and continues operating for IPv4 records.

## Development

### Project structure

```text
ipwatcher/
├── cmd/
│   └── ipwatcher/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── dnsmanager/
│   │   ├── cloudflare.go
│   │   ├── provider.go
│   │   ├── route53.go
│   │   └── types.go
│   └── ipfetcher/
│       └── ipfetcher.go
├── config.yaml.example
├── .env.example
├── install.sh
├── ipwatcher.service
└── README.md
```

### Build

```bash
go build -o ipwatcher ./cmd/ipwatcher
```

### Tests

```bash
make test-unit
make test-short
make test-coverage-unit
```

### Integration tests

Cloudflare integration tests live under `internal/dnsmanager` and require explicit credentials.

```bash
export CLOUDFLARE_API_TOKEN="your-api-token"
export CLOUDFLARE_TEST_ZONE_ID="your-zone-id"
export CLOUDFLARE_TEST_ZONE_NAME="example.com"
make test-integration
```

See [internal/dnsmanager/INTEGRATION_TESTS.md](internal/dnsmanager/INTEGRATION_TESTS.md) for details.

## License

See [LICENSE](LICENSE).

## Contributing

Pull requests are welcome.

## Acknowledgments

- [ipify](https://www.ipify.org/) for public IP discovery
- [Cloudflare](https://www.cloudflare.com/)
- [AWS Route 53](https://aws.amazon.com/route53/)

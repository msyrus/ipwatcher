# Quick Start Guide

This guide gets `ipwatcher` running quickly with either Cloudflare, AWS Route 53, or both.

## 1. Choose your provider setup

### Cloudflare

You need:

- a domain managed in Cloudflare
- an API token with `Zone -> DNS -> Edit`

Create the token at <https://dash.cloudflare.com/profile/api-tokens> using the **Edit zone DNS** template.

### AWS Route 53

You need:

- a hosted zone in Route 53
- AWS credentials with permission to list zones and update record sets
- a region value for the AWS SDK, typically `us-east-1`

## 2. Clone the repository

```bash
git clone https://github.com/msyrus/ipwatcher.git
cd ipwatcher
```

## 3. Create config and environment files

```bash
cp config.yaml.example config.yaml
cp .env.example .env
```

Edit `config.yaml` and `.env`.

### Example `config.yaml`

```yaml
refresh_rate: 0.1
sync_rate: 1
supports_ipv6: false

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

  - zone_name: "example.net"
    provider: "route53"
    records:
      - name: "home"
        type: A
```

### Example `.env`

```bash
# Cloudflare
CLOUDFLARE_API_TOKEN=your_cloudflare_token

# Route 53
AWS_ACCESS_KEY_ID=your_aws_access_key_id
AWS_SECRET_ACCESS_KEY=your_aws_secret_access_key
AWS_REGION=us-east-1
```

Only keep the variables you actually need.

## 4. Start it

### Option A: Docker Compose

```bash
mkdir -p logs
docker compose up -d
docker compose logs -f
```

### Option B: Run locally

```bash
go mod download
go build -o ipwatcher ./cmd/ipwatcher
set -a
source .env
set +a
export CONFIG_FILE=config.yaml
./ipwatcher
```

Expected startup logs look roughly like this:

```text
2026/04/16 18:00:00 Starting IP Watcher daemon...
2026/04/16 18:00:00 Current IPv4: 203.0.113.45
2026/04/16 18:00:00 DNS records for example.com (cloudflare) updated successfully
2026/04/16 18:00:00 DNS records for example.net (route53) updated successfully
```

Press `Ctrl+C` to stop the local process.

## 5. Install as a Linux service

```bash
sudo ./install.sh
sudo systemctl enable --now ipwatcher
sudo systemctl status ipwatcher
sudo journalctl -u ipwatcher -f
```

Installed files live in `/opt/ipwatcher`:

- `/opt/ipwatcher/ipwatcher`
- `/opt/ipwatcher/config.yaml`
- `/opt/ipwatcher/config.yaml.example`
- `/opt/ipwatcher/.env`
- `/opt/ipwatcher/.env.example`

## Common gotchas

### `CLOUDFLARE_API_TOKEN environment variable is required`

You have at least one `cloudflare` domain configured, but the token is missing from `.env` or your shell environment.

### `failed to create Route53 provider`

The AWS SDK could not load credentials or region settings. Confirm your `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_REGION` values.

### `AAAA record configured but supports_ipv6 is false`

Set `supports_ipv6: true` or remove the `AAAA` records.

### Records are not updating

Check that:

- `zone_name` exactly matches the authoritative zone / hosted zone name
- `name` is `@` or a relative label like `www` or `vpn`
- provider credentials have permission to edit DNS

## What next?

- Read [README.md](README.md) for full configuration details
- Use `systemd-override.example` if you want custom service overrides
- Add more zones as needed; mixed Cloudflare + Route 53 configs are supported

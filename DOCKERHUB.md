# ipwatcher

![ipwatcher logo](https://raw.githubusercontent.com/msyrus/ipwatcher/main/docs/assets/ipwatcher-logo.png)

`ipwatcher` monitors your public IP address and keeps DNS `A`/`AAAA` records in sync across **Cloudflare** and **AWS Route 53**.

Built for home labs, self-hosted apps, VPN endpoints, and servers where public IP changes are common.

---

## Why use this image?

- Automatic public IPv4 detection, with optional IPv6 support
- Supports mixed providers in one config (`cloudflare` + `route53`)
- Updates DNS immediately on IP changes
- Periodic reconciliation to correct drift
- Small container image with non-root runtime defaults
- Docker and Docker Compose friendly

---

## Quick start (Docker Compose)

```bash
git clone https://github.com/msyrus/ipwatcher.git
cd ipwatcher

cp config.yaml.example config.yaml
cp .env.example .env
mkdir -p logs

# Edit both files as needed, then run:
docker compose up -d
docker compose logs -f
```

---

## Minimal configuration example

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

  - zone_name: "example.net"
    provider: "route53"
    records:
      - name: "home"
        type: A
```

> Record names are relative labels (`@`, `www`, `vpn`), not full FQDNs.

---

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `CLOUDFLARE_API_TOKEN` | If using Cloudflare | Cloudflare token with DNS edit permissions |
| `AWS_ACCESS_KEY_ID` | Usually, if using Route 53 | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | Usually, if using Route 53 | AWS secret key |
| `AWS_SESSION_TOKEN` | Optional | AWS session token (temporary creds) |
| `AWS_REGION` | Recommended for Route 53 | Commonly `us-east-1` |
| `CONFIG_FILE` | No | Config path (default in container: `/config/config.yaml`) |

Route 53 authentication uses the AWS SDK default credential chain; env vars are the most common setup.

---

## Run without Compose

```bash
docker run -d \
  --name ipwatcher \
  --restart unless-stopped \
  -e CLOUDFLARE_API_TOKEN="your_token_here" \
  -e AWS_ACCESS_KEY_ID="your_access_key" \
  -e AWS_SECRET_ACCESS_KEY="your_secret_key" \
  -e AWS_REGION="us-east-1" \
  -v $(pwd)/config.yaml:/config/config.yaml:ro \
  -v $(pwd)/logs:/logs \
  --user 1000:1000 \
  msyrus/ipwatcher:latest
```

---

## Security defaults

The Compose setup in this project includes:

- Non-root user (`1000:1000`)
- Read-only root filesystem
- `no-new-privileges`
- Dropped Linux capabilities
- Log rotation options

---

## Common troubleshooting

- Container exits immediately: check `docker logs ipwatcher`
- DNS not updating: verify `zone_name`, record names, and provider credentials
- `AAAA` records not syncing: set `supports_ipv6: true` and ensure host IPv6 connectivity
- Config changes not applied: restart container after edits

---

## Useful links

- Source: https://github.com/msyrus/ipwatcher
- Full docs: https://github.com/msyrus/ipwatcher/blob/main/README.md
- Docker deployment guide: https://github.com/msyrus/ipwatcher/blob/main/DOCKER.md
- Releases / tags: https://github.com/msyrus/ipwatcher/releases

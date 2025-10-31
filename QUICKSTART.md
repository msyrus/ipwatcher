# Quick Start Guide

## 1. Prerequisites

- Go 1.21+ installed
- A Cloudflare account
- Your domain(s) managed by Cloudflare

## 2. Get Your Cloudflare API Token

1. Visit: <https://dash.cloudflare.com/profile/api-tokens>
2. Click **"Create Token"**
3. Use the **"Edit zone DNS"** template
4. Set permissions: Zone → DNS → Edit
5. Include your specific zone(s) or all zones
6. Create token and copy it (you'll only see it once!)

## 3. Quick Setup

### Option A: Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/msyrus/ipwatcher.git
cd ipwatcher

# Create environment file
echo "CLOUDFLARE_API_TOKEN=your_actual_token_here" > .env

# Copy and edit config
cp config.yaml.example config.yaml
nano config.yaml  # Update zone_name and records

# Start with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f
```

### Option B: Build from Source

```bash
# Clone the repository
git clone https://github.com/msyrus/ipwatcher.git
cd ipwatcher

# Download dependencies
go mod download

# Create environment file
echo "CLOUDFLARE_API_TOKEN=your_actual_token_here" > .env

# Copy and edit config
cp config.yaml.example config.yaml
nano config.yaml  # Update zone_name and records
```

## 4. Test Run

```bash
# Build and run
go build -o ipwatcher ./cmd/ipwatcher
./ipwatcher
```

You should see output like:

```text
2024/10/30 18:00:00 Starting IP Watcher daemon...
2024/10/30 18:00:00 Current IPv4: 203.0.113.45
2024/10/30 18:00:00 Current IPv6: 2001:db8::1
2024/10/30 18:00:00 Updated DNS record example.com (A) to IP 203.0.113.45
```

Press `Ctrl+C` to stop.

## 5. Install as Service (Linux)

```bash
# Run installation script
chmod +x install.sh
sudo ./install.sh

# Edit the installed config
sudo nano /opt/ipwatcher/config.yaml
sudo nano /opt/ipwatcher/.env

# Start the service
sudo systemctl enable ipwatcher
sudo systemctl start ipwatcher

# Check status
sudo systemctl status ipwatcher

# View logs
sudo journalctl -u ipwatcher -f
```

## Example Configuration

```yaml
refresh_rate: 0.1    # Check IP every 10 seconds
sync_rate: 1         # Verify DNS every minute
supports_ipv6: false # Set to true if your network supports IPv6

domains:
  - zone_name: "example.com"
    records:
      # Root domain
      - name: "@"
        type: A
        proxied: false

      # WWW subdomain (proxied through Cloudflare)
      - name: "www"
        type: A
        proxied: true

      # API subdomain (direct to origin)
      - name: "api"
        type: A
        proxied: false
```

## Troubleshooting

### "CLOUDFLARE_API_TOKEN environment variable is required"

- Make sure you have a `.env` file with your token
- Or export it: `export CLOUDFLARE_API_TOKEN="your_token"`

### "Failed to load configuration"

- Check that `config.yaml` exists
- Validate YAML syntax using a [YAML linter](https://www.yamllint.com/)

### DNS records not updating

- Verify your API token has DNS edit permissions
- Check zone_name matches your domain exactly
- Ensure record names use correct format ("@" for root, "www" for subdomain)

## Next Steps

- See [README.md](README.md) for full documentation
- Adjust refresh_rate and sync_rate based on your needs
- Add more domains and subdomains to your config
- Set up monitoring for the systemd service

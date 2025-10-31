# Docker Deployment Guide

This guide explains how to run ipwatcher using Docker and Docker Compose.

## Quick Start with Docker Compose

### 1. Prerequisites

- Docker installed
- Docker Compose installed
- Cloudflare API token
- Configuration file ready

### 2. Setup

```bash
# Clone the repository
git clone https://github.com/msyrus/ipwatcher.git
cd ipwatcher

# Create configuration file
cp config.yaml.example config.yaml
nano config.yaml  # Edit with your settings

# Create .env file with your Cloudflare token
echo "CLOUDFLARE_API_TOKEN=your_token_here" > .env

# Create logs directory
mkdir -p logs
```

### 3. Run with Docker Compose

```bash
# Build and start the container
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the container
docker-compose down
```

## Manual Docker Commands

### Build the Image

```bash
docker build -t ipwatcher:latest .
```

### Run the Container

```bash
docker run -d \
  --name ipwatcher \
  --restart unless-stopped \
  -e CLOUDFLARE_API_TOKEN="your_token_here" \
  -v $(pwd)/config.yaml:/config/config.yaml:ro \
  -v $(pwd)/logs:/logs \
  --user 1000:1000 \
  ipwatcher:latest
```

### View Logs

```bash
docker logs -f ipwatcher
```

### Stop and Remove

```bash
docker stop ipwatcher
docker rm ipwatcher
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `CLOUDFLARE_API_TOKEN` | Yes | Your Cloudflare API token |
| `CONFIG_FILE` | No | Path to config file (default: `/config/config.yaml`) |

### Volume Mounts

| Host Path | Container Path | Purpose |
|-----------|----------------|---------|
| `./config.yaml` | `/config/config.yaml` | Configuration file |
| `./logs` | `/logs` | Log directory (optional) |

## Advanced Usage

### Custom User/Group

By default, the container runs as user `1000:1000`. To use a different user:

```bash
# Get your user ID and group ID
id -u  # Returns your UID
id -g  # Returns your GID

# Run with your user
docker run -d \
  --name ipwatcher \
  --user $(id -u):$(id -g) \
  -e CLOUDFLARE_API_TOKEN="your_token" \
  -v $(pwd)/config.yaml:/config/config.yaml:ro \
  ipwatcher:latest
```

### Docker Compose Override

Create `docker-compose.override.yml` for local customization:

```yaml
version: '3.8'

services:
  ipwatcher:
    # Override user
    user: "1001:1001"

    # Add custom environment variables
    environment:
      - TZ=America/New_York

    # Add additional volumes
    volumes:
      - ./custom-config.yaml:/config/config.yaml:ro
```

### Health Check

The container includes a health check that runs every 60 seconds:

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' ipwatcher

# View health check logs
docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' ipwatcher
```

## Building for Different Architectures

### ARM64 (e.g., Raspberry Pi, Apple Silicon)

```bash
docker build --platform linux/arm64 -t ipwatcher:arm64 .
```

### Multi-architecture Build

```bash
# Create a builder
docker buildx create --name multiarch --use

# Build for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t ipwatcher:latest \
  --push \
  .
```

## Troubleshooting

### Container Exits Immediately

Check logs for errors:

```bash
docker logs ipwatcher
```

Common issues:

- Missing `CLOUDFLARE_API_TOKEN` environment variable
- Invalid configuration file
- Configuration file not mounted correctly

### Permission Denied Errors

Ensure the user running the container has permission to read the config file:

```bash
chmod 644 config.yaml
```

Or adjust the `--user` flag to match your file ownership.

### Cannot Fetch IP Address

Ensure the container has network access:

```bash
# Test network from inside container
docker exec ipwatcher wget -O- https://api.ipify.org
```

### Configuration Changes Not Applied

The configuration is mounted read-only. After changing `config.yaml`:

```bash
# Restart the container
docker-compose restart
# or
docker restart ipwatcher
```

## Security Best Practices

1. **Run as non-root user**: Always use `--user` flag
2. **Read-only root filesystem**: Included in docker-compose.yml
3. **Drop all capabilities**: Included in docker-compose.yml
4. **No new privileges**: Included in docker-compose.yml
5. **Secrets management**: Use Docker secrets or environment files
6. **Network isolation**: Use bridge network unless host network is required

### Using Docker Secrets (Swarm Mode)

```bash
# Create secret
echo "your_token_here" | docker secret create cloudflare_token -

# Update docker-compose.yml
services:
  ipwatcher:
    secrets:
      - cloudflare_token
    environment:
      - CLOUDFLARE_API_TOKEN_FILE=/run/secrets/cloudflare_token

secrets:
  cloudflare_token:
    external: true
```

## Container Registry

### Push to Docker Hub

```bash
# Tag the image
docker tag ipwatcher:latest yourusername/ipwatcher:latest

# Push to Docker Hub
docker push yourusername/ipwatcher:latest
```

### Pull and Run

```bash
docker pull yourusername/ipwatcher:latest

docker run -d \
  --name ipwatcher \
  -e CLOUDFLARE_API_TOKEN="your_token" \
  -v $(pwd)/config.yaml:/config/config.yaml:ro \
  yourusername/ipwatcher:latest
```

## Monitoring and Maintenance

### View Resource Usage

```bash
# Real-time stats
docker stats ipwatcher

# Detailed inspection
docker inspect ipwatcher
```

### Log Management

Configure log rotation in docker-compose.yml (already included):

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

### Automatic Updates

Use Watchtower for automatic container updates:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower \
  --interval 3600 \
  ipwatcher
```

## Example Production Deployment

Complete `docker-compose.yml` for production:

```yaml
version: '3.8'

services:
  ipwatcher:
    image: ipwatcher:latest
    container_name: ipwatcher
    restart: always
    user: "1000:1000"

    environment:
      - CLOUDFLARE_API_TOKEN=${CLOUDFLARE_API_TOKEN}
      - CONFIG_FILE=/config/config.yaml
      - TZ=UTC

    volumes:
      - ./config.yaml:/config/config.yaml:ro
      - logs:/logs

    security_opt:
      - no-new-privileges:true

    read_only: true

    cap_drop:
      - ALL

    networks:
      - ipwatcher-net

    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "5"

    healthcheck:
      test: ["CMD-SHELL", "pgrep ipwatcher || exit 1"]
      interval: 60s
      timeout: 10s
      retries: 3
      start_period: 30s

volumes:
  logs:
    driver: local

networks:
  ipwatcher-net:
    driver: bridge
```

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Best Practices for Writing Dockerfiles](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)

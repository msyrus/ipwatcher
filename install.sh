#!/bin/bash

# Installation script for IP Watcher daemon

set -e

echo "Installing IP Watcher daemon..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

# Create user and group if they don't exist
if ! id -u ipwatcher > /dev/null 2>&1; then
    echo "Creating ipwatcher user..."
    useradd --system --no-create-home --shell /bin/false ipwatcher
fi

# Create installation directory
INSTALL_DIR="/opt/ipwatcher"
echo "Creating installation directory: $INSTALL_DIR"
mkdir -p $INSTALL_DIR
mkdir -p $INSTALL_DIR/logs

# Build the binary
echo "Building ipwatcher binary..."
go build -o ipwatcher ./cmd/ipwatcher

# Copy files
echo "Installing files..."
cp ipwatcher $INSTALL_DIR/
cp config.yaml.example $INSTALL_DIR/config.yaml
cp .env.example $INSTALL_DIR/.env 2>/dev/null || echo "CLOUDFLARE_API_TOKEN=" > $INSTALL_DIR/.env

# Set permissions
echo "Setting permissions..."
chown -R ipwatcher:ipwatcher $INSTALL_DIR
chmod 755 $INSTALL_DIR/ipwatcher
chmod 644 $INSTALL_DIR/config.yaml
chmod 600 $INSTALL_DIR/.env

# Install systemd service
echo "Installing systemd service..."
cp ipwatcher.service /etc/systemd/system/
systemctl daemon-reload

echo ""
echo "Installation complete!"
echo ""
echo "Next steps:"
echo "1. Edit configuration: sudo nano $INSTALL_DIR/config.yaml"
echo "2. Set Cloudflare API token: sudo nano $INSTALL_DIR/.env"
echo "3. Enable service: sudo systemctl enable ipwatcher"
echo "4. Start service: sudo systemctl start ipwatcher"
echo "5. Check status: sudo systemctl status ipwatcher"
echo "6. View logs: sudo journalctl -u ipwatcher -f"

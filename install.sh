#!/usr/bin/env bash

# Installation script for the ipwatcher daemon.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/opt/ipwatcher"
SERVICE_FILE="/etc/systemd/system/ipwatcher.service"

echo "Installing ipwatcher..."

if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)."
    exit 1
fi

cd "$SCRIPT_DIR"

if ! command -v go >/dev/null 2>&1; then
    echo "Go is required to build ipwatcher during installation."
    exit 1
fi

if ! id -u ipwatcher >/dev/null 2>&1; then
    echo "Creating ipwatcher system user..."
    useradd --system --no-create-home --shell /usr/sbin/nologin ipwatcher
fi

echo "Creating installation directory at $INSTALL_DIR"
install -d -m 755 "$INSTALL_DIR"

echo "Building ipwatcher binary..."
go build -o "$SCRIPT_DIR/ipwatcher" ./cmd/ipwatcher

echo "Installing binary and configuration files..."
install -m 755 "$SCRIPT_DIR/ipwatcher" "$INSTALL_DIR/ipwatcher"
install -m 644 "$SCRIPT_DIR/config.yaml.example" "$INSTALL_DIR/config.yaml.example"
install -m 600 "$SCRIPT_DIR/.env.example" "$INSTALL_DIR/.env.example"

if [ -f "$SCRIPT_DIR/config.yaml" ]; then
    install -m 640 "$SCRIPT_DIR/config.yaml" "$INSTALL_DIR/config.yaml"
elif [ ! -f "$INSTALL_DIR/config.yaml" ]; then
    install -m 640 "$SCRIPT_DIR/config.yaml.example" "$INSTALL_DIR/config.yaml"
fi

if [ -f "$SCRIPT_DIR/.env" ]; then
    install -m 600 "$SCRIPT_DIR/.env" "$INSTALL_DIR/.env"
elif [ ! -f "$INSTALL_DIR/.env" ]; then
    install -m 600 "$SCRIPT_DIR/.env.example" "$INSTALL_DIR/.env"
fi

echo "Installing systemd service..."
install -m 644 "$SCRIPT_DIR/ipwatcher.service" "$SERVICE_FILE"

chown -R ipwatcher:ipwatcher "$INSTALL_DIR"
systemctl daemon-reload

echo
echo "Installation complete."
echo
echo "Installed files:"
echo "  - $INSTALL_DIR/ipwatcher"
echo "  - $INSTALL_DIR/config.yaml"
echo "  - $INSTALL_DIR/config.yaml.example"
echo "  - $INSTALL_DIR/.env"
echo "  - $INSTALL_DIR/.env.example"
echo
echo "Next steps:"
echo "1. Review configuration: sudo nano $INSTALL_DIR/config.yaml"
echo "2. Add provider credentials: sudo nano $INSTALL_DIR/.env"
echo "3. Enable the service: sudo systemctl enable ipwatcher"
echo "4. Start the service: sudo systemctl start ipwatcher"
echo "5. Check status: sudo systemctl status ipwatcher"
echo "6. Tail logs: sudo journalctl -u ipwatcher -f"

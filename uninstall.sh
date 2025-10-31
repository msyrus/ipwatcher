#!/bin/bash

# Uninstallation script for IP Watcher daemon

set -e

echo "Uninstalling IP Watcher daemon..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

# Stop and disable service
if systemctl is-active --quiet ipwatcher; then
    echo "Stopping ipwatcher service..."
    systemctl stop ipwatcher
fi

if systemctl is-enabled --quiet ipwatcher; then
    echo "Disabling ipwatcher service..."
    systemctl disable ipwatcher
fi

# Remove systemd service file
if [ -f /etc/systemd/system/ipwatcher.service ]; then
    echo "Removing systemd service file..."
    rm /etc/systemd/system/ipwatcher.service
    systemctl daemon-reload
fi

# Remove installation directory
INSTALL_DIR="/opt/ipwatcher"
if [ -d "$INSTALL_DIR" ]; then
    echo "Removing installation directory: $INSTALL_DIR"
    read -p "Do you want to remove config and logs? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf $INSTALL_DIR
    else
        # Keep config but remove binary
        rm -f $INSTALL_DIR/ipwatcher
        echo "Binary removed. Config and logs preserved at $INSTALL_DIR"
    fi
fi

# Optionally remove user
read -p "Do you want to remove the ipwatcher user? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if id -u ipwatcher > /dev/null 2>&1; then
        echo "Removing ipwatcher user..."
        userdel ipwatcher
    fi
fi

echo ""
echo "Uninstallation complete!"

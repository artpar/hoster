#!/bin/bash
# Hoster Production Setup Script
# Run this script on a fresh Ubuntu/Debian server

set -e

echo "=========================================="
echo "  Hoster Production Setup"
echo "=========================================="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (sudo ./setup.sh)"
    exit 1
fi

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo "Cannot detect OS"
    exit 1
fi

echo "[1/8] Updating system packages..."
apt-get update
apt-get upgrade -y

echo "[2/8] Installing dependencies..."
apt-get install -y \
    curl \
    wget \
    git \
    jq \
    ufw \
    fail2ban \
    unattended-upgrades

echo "[3/8] Installing Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker
    systemctl start docker
else
    echo "Docker already installed"
fi

echo "[4/8] Creating service users..."
# Create hoster user
if ! id "hoster" &>/dev/null; then
    useradd -r -s /bin/false -d /opt/hoster hoster
    usermod -aG docker hoster
fi

# Create apigate user
if ! id "apigate" &>/dev/null; then
    useradd -r -s /bin/false -d /opt/apigate apigate
fi

echo "[5/8] Creating directories..."
mkdir -p /opt/hoster/bin
mkdir -p /opt/apigate
mkdir -p /var/lib/hoster
mkdir -p /var/lib/apigate
mkdir -p /etc/hoster
mkdir -p /var/backups/hoster

# Set permissions
chown -R hoster:hoster /opt/hoster /var/lib/hoster
chown -R apigate:apigate /opt/apigate /var/lib/apigate
chmod 750 /var/lib/hoster /var/lib/apigate

echo "[6/8] Configuring firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
# Allow app proxy port range (optional - comment out if using reverse proxy)
# ufw allow 30000:39999/tcp
ufw --force enable

echo "[7/8] Configuring automatic security updates..."
cat > /etc/apt/apt.conf.d/20auto-upgrades << EOF
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
EOF

echo "[8/8] Installing systemd services..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(dirname "$SCRIPT_DIR")"

cp "$DEPLOY_DIR/systemd/hoster.service" /etc/systemd/system/
cp "$DEPLOY_DIR/systemd/apigate.service" /etc/systemd/system/
systemctl daemon-reload

echo ""
echo "=========================================="
echo "  Setup Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Download and install binaries:"
echo "   - Place hoster binary at: /opt/hoster/bin/hoster"
echo "   - Place apigate binary at: /opt/apigate/apigate"
echo ""
echo "2. Configure environment:"
echo "   cp $DEPLOY_DIR/env.example /etc/hoster/.env"
echo "   nano /etc/hoster/.env"
echo ""
echo "3. Configure DNS and SSL certificates"
echo ""
echo "4. Start services:"
echo "   systemctl enable apigate hoster"
echo "   systemctl start apigate hoster"
echo ""
echo "5. Check status:"
echo "   systemctl status apigate hoster"
echo ""

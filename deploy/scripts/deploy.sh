#!/bin/bash
# Hoster Deployment/Update Script
# Updates Hoster to the latest version

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "=========================================="
echo "  Hoster Deployment"
echo "=========================================="
echo "Project directory: $PROJECT_DIR"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (sudo ./deploy.sh)"
    exit 1
fi

# Optional: Pull latest code
if [ "$1" == "--pull" ]; then
    echo "[1/5] Pulling latest code..."
    cd "$PROJECT_DIR"
    git pull
else
    echo "[1/5] Skipping git pull (use --pull to update code)"
fi

echo "[2/5] Building Hoster..."
cd "$PROJECT_DIR"

# Build for production
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/hoster-linux-amd64 ./cmd/hoster

echo "[3/5] Building Frontend..."
cd "$PROJECT_DIR/web"
npm ci --production=false
npm run build

echo "[4/5] Deploying binaries..."
# Stop services
systemctl stop hoster || true

# Copy binary
cp "$PROJECT_DIR/bin/hoster-linux-amd64" /opt/hoster/bin/hoster
chmod +x /opt/hoster/bin/hoster
chown hoster:hoster /opt/hoster/bin/hoster

# Copy frontend build
rm -rf /opt/hoster/web/dist
mkdir -p /opt/hoster/web
cp -r "$PROJECT_DIR/web/dist" /opt/hoster/web/
chown -R hoster:hoster /opt/hoster/web

echo "[5/5] Starting services..."
systemctl start hoster

# Wait for startup
sleep 3

# Health check
echo ""
echo "Checking service health..."
if curl -s http://localhost:8080/health | grep -q "healthy"; then
    echo "Hoster: OK"
else
    echo "Hoster: FAILED"
    echo "Check logs: journalctl -u hoster -n 50"
    exit 1
fi

echo ""
echo "=========================================="
echo "  Deployment Complete!"
echo "=========================================="
echo ""
echo "Service status:"
systemctl status hoster --no-pager | head -10

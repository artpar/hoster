# Hoster Production Deployment

This directory contains all configuration files, scripts, and documentation for deploying Hoster to production.

## Architecture Overview

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                    Production Server                      │
                    │                                                           │
Internet ──────────►│  ┌─────────────┐    ┌─────────────┐    ┌──────────────┐ │
    :443            │  │   APIGate   │───►│   Hoster    │───►│    Docker    │ │
    :80             │  │   (auth/    │    │   (API +    │    │  Containers  │ │
                    │  │   billing)  │    │   Proxy)    │    │  (user apps) │ │
                    │  │   :8082     │    │ :8080/:9091 │    │ 30000-39999  │ │
                    │  └─────────────┘    └─────────────┘    └──────────────┘ │
                    │         │                  │                             │
                    │         ▼                  ▼                             │
                    │  ┌─────────────┐    ┌─────────────┐                     │
                    │  │  apigate.db │    │  hoster.db  │                     │
                    │  └─────────────┘    └─────────────┘                     │
                    └─────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Server Requirements

- Ubuntu 22.04+ or Debian 12+
- 2+ CPU cores, 4GB+ RAM
- Docker installed
- Domain with DNS configured

### 2. Initial Setup

```bash
# Clone and setup
git clone https://github.com/artpar/hoster.git
cd hoster/deploy

# Run initial setup
chmod +x scripts/*.sh
./scripts/setup.sh
```

### 3. Configure Environment

```bash
# Copy and edit environment file
cp env.example /etc/hoster/.env
nano /etc/hoster/.env
```

### 4. Start Services

```bash
# Enable and start services
sudo systemctl enable apigate hoster
sudo systemctl start apigate hoster

# Check status
sudo systemctl status apigate hoster
```

## Directory Structure

```
deploy/
├── README.md                 # This file
├── docker-compose.prod.yml   # Production Docker Compose (optional)
├── env.example               # Environment variables template
├── systemd/
│   ├── apigate.service       # APIGate systemd unit
│   └── hoster.service        # Hoster systemd unit
├── scripts/
│   ├── setup.sh              # Initial server setup
│   ├── deploy.sh             # Deploy/update script
│   └── backup.sh             # Database backup script
└── docs/
    ├── ssl-setup.md          # SSL certificate setup guide
    ├── dns-setup.md          # DNS configuration guide
    └── troubleshooting.md    # Common issues and solutions
```

## Environment Variables

See `env.example` for all available configuration options. Key variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `HOSTER_SERVER_PORT` | API server port | `8080` |
| `HOSTER_BILLING_ENABLED` | Enable billing | `true` |
| `HOSTER_APIGATE_URL` | APIGate base URL | `http://localhost:8082` |
| `HOSTER_APP_PROXY_ENABLED` | Enable app proxy | `true` |
| `HOSTER_APP_PROXY_BASE_DOMAIN` | Base domain for apps | `apps.example.com` |

## DNS Configuration

Required DNS records:

| Type | Name | Value | Purpose |
|------|------|-------|---------|
| A | `@` or `api` | `<server-ip>` | API endpoint |
| A | `portal` | `<server-ip>` | User portal |
| A | `*.apps` | `<server-ip>` | App proxy wildcard |

## SSL Certificates

Use Let's Encrypt with certbot:

```bash
# Install certbot
sudo apt install certbot

# Get wildcard certificate (requires DNS challenge)
sudo certbot certonly --manual --preferred-challenges dns \
  -d example.com -d *.example.com -d *.apps.example.com
```

## Updating

```bash
# Pull latest changes
cd /opt/hoster
git pull

# Rebuild and restart
./deploy/scripts/deploy.sh
```

## Backup

```bash
# Manual backup
./deploy/scripts/backup.sh

# Setup daily backup cron
echo "0 2 * * * /opt/hoster/deploy/scripts/backup.sh" | sudo crontab -
```

## Monitoring

Check service status:
```bash
sudo systemctl status apigate hoster
```

View logs:
```bash
sudo journalctl -u hoster -f
sudo journalctl -u apigate -f
```

Health endpoints:
```bash
curl http://localhost:8080/health  # Hoster
curl http://localhost:8082/health  # APIGate
curl http://localhost:9091/health  # App Proxy
```

## Troubleshooting

See `docs/troubleshooting.md` for common issues and solutions.

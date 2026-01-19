# SSL Certificate Setup

This guide covers setting up SSL certificates for Hoster using Let's Encrypt.

## Prerequisites

- Domain name pointed to your server
- Ports 80 and 443 accessible from the internet
- `certbot` installed

## Installation

### Ubuntu/Debian

```bash
sudo apt update
sudo apt install certbot
```

## Certificate Types

### 1. Single Domain Certificate

For a simple setup with just the API domain:

```bash
sudo certbot certonly --standalone -d api.example.com
```

### 2. Multiple Domains

For API, portal, and app proxy:

```bash
sudo certbot certonly --standalone \
  -d example.com \
  -d api.example.com \
  -d portal.example.com
```

### 3. Wildcard Certificate (Recommended)

For the app proxy wildcard domain, you need a DNS challenge:

```bash
sudo certbot certonly --manual --preferred-challenges dns \
  -d example.com \
  -d "*.example.com" \
  -d "*.apps.example.com"
```

When prompted, add the TXT records to your DNS:
1. `_acme-challenge.example.com`
2. `_acme-challenge.apps.example.com`

Wait for DNS propagation before continuing.

## Certificate Location

Certificates are stored in:
```
/etc/letsencrypt/live/example.com/
├── fullchain.pem   # Certificate + intermediate
├── privkey.pem     # Private key
├── cert.pem        # Certificate only
└── chain.pem       # Intermediate certificates
```

## Auto-Renewal

Certbot automatically installs a renewal timer. Verify it's working:

```bash
sudo systemctl status certbot.timer
sudo certbot renew --dry-run
```

## Using with Nginx

Example nginx configuration with SSL:

```nginx
server {
    listen 80;
    server_name example.com *.example.com *.apps.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

server {
    listen 443 ssl http2;
    server_name *.apps.example.com;

    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:9091;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Troubleshooting

### DNS not propagated
Wait 5-10 minutes and verify with:
```bash
dig TXT _acme-challenge.example.com
```

### Port 80 in use
Stop the service using port 80 temporarily:
```bash
sudo systemctl stop nginx
sudo certbot certonly --standalone -d example.com
sudo systemctl start nginx
```

### Certificate not renewing
Check certbot logs:
```bash
sudo journalctl -u certbot
```

# Troubleshooting Guide

Common issues and solutions for Hoster production deployments.

## Service Won't Start

### Check service status
```bash
sudo systemctl status hoster
sudo systemctl status apigate
```

### View recent logs
```bash
sudo journalctl -u hoster -n 100 --no-pager
sudo journalctl -u apigate -n 100 --no-pager
```

### Common causes

1. **Missing environment file**
   ```bash
   ls -la /etc/hoster/.env
   # If missing, copy from template
   cp /opt/hoster/deploy/env.example /etc/hoster/.env
   ```

2. **Permission issues**
   ```bash
   chown -R hoster:hoster /opt/hoster /var/lib/hoster
   chown -R apigate:apigate /opt/apigate /var/lib/apigate
   ```

3. **Port already in use**
   ```bash
   sudo lsof -i :8080
   sudo lsof -i :8082
   sudo lsof -i :9091
   ```

4. **Docker socket permissions**
   ```bash
   usermod -aG docker hoster
   # Restart service after adding to group
   ```

## Health Check Failures

### Test endpoints directly
```bash
curl -v http://localhost:8080/health
curl -v http://localhost:8082/health
curl -v http://localhost:9091/health
```

### Expected responses
```json
{"status":"healthy"}           // Hoster
{"status":"ok"}                // APIGate
{"status":"ok","deployments_routable":0,"base_domain":"apps.example.com"}  // App Proxy
```

## Database Issues

### Database locked
SQLite "database is locked" error:
```bash
# Stop services
sudo systemctl stop hoster apigate

# Check for stale processes
fuser /var/lib/hoster/hoster.db
fuser /var/lib/apigate/apigate.db

# Restart services
sudo systemctl start apigate hoster
```

### Database corruption
```bash
# Backup current database
cp /var/lib/hoster/hoster.db /var/lib/hoster/hoster.db.bak

# Check integrity
sqlite3 /var/lib/hoster/hoster.db "PRAGMA integrity_check;"

# If corrupted, restore from backup
cp /var/backups/hoster/hoster_*.db /var/lib/hoster/hoster.db
```

## APIGate Integration Issues

### Auto-registration fails
```bash
# Check logs for registration errors
grep -i "register\|apigate" /var/log/hoster/hoster.log

# Verify APIGate is accessible
curl http://localhost:8082/health

# Test admin API access
curl http://localhost:8082/admin/upstreams \
  -H "X-API-Key: YOUR_API_KEY"
```

### Billing events not sending
```bash
# Check billing configuration
grep BILLING /etc/hoster/.env

# Look for billing errors in logs
grep -i "billing\|meter" /var/log/hoster/hoster.log
```

## App Proxy Issues

### Deployments not accessible

1. **Check deployment status**
   ```bash
   curl http://localhost:8080/api/v1/deployments | jq '.data[] | {name, status, proxy_port}'
   ```

2. **Verify proxy routing**
   ```bash
   curl -H "Host: myapp.apps.example.com" http://localhost:9091
   ```

3. **Check container is running**
   ```bash
   docker ps | grep myapp
   ```

4. **Check proxy port binding**
   ```bash
   sudo lsof -i :30000  # Check specific port
   ```

### Wildcard DNS not resolving
```bash
# Test DNS resolution
dig +short myapp.apps.example.com

# Should return your server IP
# If not, check DNS configuration
```

## Docker Issues

### Container won't start
```bash
# Check Docker logs
docker logs <container_id>

# Check Docker daemon
sudo systemctl status docker
sudo journalctl -u docker -n 50
```

### Disk space issues
```bash
# Check disk usage
df -h

# Clean up Docker
docker system prune -a
```

## Performance Issues

### High CPU usage
```bash
# Check top processes
top -c

# Check Docker stats
docker stats
```

### High memory usage
```bash
# Check memory
free -h

# Check for memory leaks
cat /proc/$(pgrep hoster)/status | grep Vm
```

## Firewall Issues

### Check firewall rules
```bash
sudo ufw status verbose
```

### Required ports
- 80/443: HTTP/HTTPS (via reverse proxy)
- 8080: Hoster API (internal)
- 8082: APIGate (internal)
- 9091: App Proxy (internal or external)
- 30000-39999: Deployment ports (if exposed)

## Getting Help

1. Check logs first: `journalctl -u hoster -f`
2. Search existing issues: https://github.com/artpar/hoster/issues
3. Create new issue with:
   - Error messages
   - Service status output
   - Relevant configuration (redact secrets)
   - Steps to reproduce

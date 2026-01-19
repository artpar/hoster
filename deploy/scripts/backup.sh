#!/bin/bash
# Hoster Backup Script
# Creates backups of databases and configuration

set -e

BACKUP_DIR="/var/backups/hoster"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=7

echo "=========================================="
echo "  Hoster Backup - $DATE"
echo "=========================================="

# Create backup directory
mkdir -p "$BACKUP_DIR"

echo "[1/4] Backing up Hoster database..."
if [ -f /var/lib/hoster/hoster.db ]; then
    sqlite3 /var/lib/hoster/hoster.db ".backup '$BACKUP_DIR/hoster_$DATE.db'"
    echo "  -> $BACKUP_DIR/hoster_$DATE.db"
else
    echo "  -> Skipped (database not found)"
fi

echo "[2/4] Backing up APIGate database..."
if [ -f /var/lib/apigate/apigate.db ]; then
    sqlite3 /var/lib/apigate/apigate.db ".backup '$BACKUP_DIR/apigate_$DATE.db'"
    echo "  -> $BACKUP_DIR/apigate_$DATE.db"
else
    echo "  -> Skipped (database not found)"
fi

echo "[3/4] Backing up configuration..."
if [ -f /etc/hoster/.env ]; then
    cp /etc/hoster/.env "$BACKUP_DIR/hoster_env_$DATE"
    echo "  -> $BACKUP_DIR/hoster_env_$DATE"
else
    echo "  -> Skipped (config not found)"
fi

echo "[4/4] Cleaning old backups (older than $RETENTION_DAYS days)..."
find "$BACKUP_DIR" -type f -mtime +$RETENTION_DAYS -delete
echo "  -> Done"

# Create compressed archive
echo ""
echo "Creating compressed archive..."
ARCHIVE="$BACKUP_DIR/hoster_backup_$DATE.tar.gz"
tar -czf "$ARCHIVE" -C "$BACKUP_DIR" \
    "hoster_$DATE.db" \
    "apigate_$DATE.db" \
    "hoster_env_$DATE" \
    2>/dev/null || true

# Cleanup individual files
rm -f "$BACKUP_DIR/hoster_$DATE.db" \
      "$BACKUP_DIR/apigate_$DATE.db" \
      "$BACKUP_DIR/hoster_env_$DATE" \
      2>/dev/null || true

echo ""
echo "=========================================="
echo "  Backup Complete!"
echo "=========================================="
echo ""
echo "Archive: $ARCHIVE"
echo "Size: $(du -h "$ARCHIVE" | cut -f1)"
echo ""
echo "To restore:"
echo "  tar -xzf $ARCHIVE -C /tmp"
echo "  sqlite3 /var/lib/hoster/hoster.db \".restore '/tmp/hoster_$DATE.db'\""
echo ""

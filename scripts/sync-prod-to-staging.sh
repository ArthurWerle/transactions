#!/usr/bin/env bash
#
# Nightly sync: dump production Postgres → restore into staging Postgres.
# Intended to run as a cron job on the Docker host.
#
# crontab entry (runs at 02:00 every night):
#   0 2 * * * /path/to/sync-prod-to-staging.sh >> /var/log/sync-prod-to-staging.log 2>&1
#
set -euo pipefail

PROD_CONTAINER="transactions_postgres"
STAGING_CONTAINER="transactions_postgres_staging"
DB_NAME="transactions"
DB_USER="postgres"

DUMP_FILE="/tmp/transactions_prod_dump_$(date +%Y%m%d).sql"

echo "[$(date)] Starting prod → staging sync"

# 1. Dump production database
docker exec "$PROD_CONTAINER" \
  pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists \
  > "$DUMP_FILE"

echo "[$(date)] Dump completed: $(wc -c < "$DUMP_FILE") bytes"

# 2. Restore into staging (drop + recreate to get a clean state)
docker exec -i "$STAGING_CONTAINER" \
  psql -U "$DB_USER" -d "$DB_NAME" \
  < "$DUMP_FILE"

echo "[$(date)] Restore completed"

# 3. Cleanup
rm -f "$DUMP_FILE"

echo "[$(date)] Sync finished successfully"

#!/usr/bin/env bash
set -euo pipefail

# If CRON_SCHEDULE is provided, run on a schedule via supercronic.
# Otherwise, run the backup once and exit (one-shot mode).

if [[ -n "${CRON_SCHEDULE:-}" ]]; then
  CRONTAB_PATH="/tmp/crontab"
  echo "${CRON_SCHEDULE} /usr/local/bin/pg-backup-service" > "${CRONTAB_PATH}"
  echo "Starting supercronic with schedule: ${CRON_SCHEDULE}"
  exec /usr/local/bin/supercronic "${CRONTAB_PATH}"
fi

echo "CRON_SCHEDULE not set — running backup once."
exec /usr/local/bin/pg-backup-service

# Scheduling pg-backup-service

There are two ways to schedule recurring backups. Pick whichever fits your environment.

## Option A — Docker (recommended)

The Docker image has a cron scheduler ([supercronic](https://github.com/aptible/supercronic)) built in. Set `CRON_SCHEDULE` in `.env` and the container handles the rest — no host crontab needed. See [docker.md](./docker.md) for full setup.

Quickstart:

```bash
cp .env.example .env       # set CRON_SCHEDULE, e.g. "30 20 * * *"
docker compose up -d
```

## Option B — Host cron with the binary

If you're running the standalone binary on a VPS instead of Docker, use the host's crontab. To run the backup every day at 20:30 UTC:

### 1. Preparation

Ensure the binary is built (or downloaded from a Release) and you have a `.env` file in the same directory.

### 2. Create a run script

Create `run_backup.sh` next to the binary:

```bash
#!/bin/bash
cd /path/to/pg-backup-service
set -a
source .env
set +a
./pg-backup-service
```

Make it executable:

```bash
chmod +x run_backup.sh
```

### 3. Add to crontab

Open the crontab editor:

```bash
crontab -e
```

If your server uses UTC:

```cron
30 20 * * * /path/to/pg-backup-service/run_backup.sh >> /path/to/pg-backup-service/backup.log 2>&1
```

If your server uses Indian Standard Time (IST), the equivalent of 20:30 UTC is 02:00 IST:

```cron
0 2 * * * /path/to/pg-backup-service/run_backup.sh >> /path/to/pg-backup-service/backup.log 2>&1
```

### Note on environment variables

The cron environment is minimal. The wrapper script above sources `.env` so all required variables (`PG_PASSWORD`, `ZDRIVE_KEY`, `ZDRIVE_SECRET`, etc.) are loaded before the binary runs.

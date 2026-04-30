# PostgreSQL Backup Service

A Go-based service designed to automate full PostgreSQL database dumps and securely upload them to [ZDrive](https://ziqx.cc/drive).

## Quick Start (VPS)

### 1. Direct Download

You can download the pre-built binary directly to your VPS:

```bash
# Using wget
wget https://github.com/fathah/pg-backup-service/releases/latest/download/pg-backup-service && chmod +x pg-backup-service

# Using curl
curl -L -O https://github.com/fathah/pg-backup-service/releases/latest/download/pg-backup-service && chmod +x pg-backup-service
```

### 2. Configuration

Create a `.env` file or export environment variables:

| Variable        | Description                    | Default     |
| --------------- | ------------------------------ | ----------- |
| `PG_HOST`       | PostgreSQL server host         | `localhost` |
| `PG_PORT`       | PostgreSQL server port         | `5432`      |
| `PG_USER`       | PostgreSQL user                | `postgres`  |
| `PG_PASSWORD`   | PostgreSQL password (Required) | -           |
| `ZDRIVE_KEY`    | Your ZDrive ID (Required)      | -           |
| `ZDRIVE_SECRET` | Your ZDrive secret (Required)  | -           |
| `BACKUP_PREFIX` | Prefix for the backup filename | `pg_backup` |

### 3. Running

```bash
./pg-backup-service
```

### 4. Daily Backups (Cron)

For host-cron setup with the binary, see [cronjob.md](./cronjob.md). If you'd rather skip host cron entirely, the Docker image below has a scheduler built in.

---

## Docker (recommended)

A pre-built Docker image is published to GitHub Container Registry. It bundles the PostgreSQL client tools and a built-in cron scheduler ([supercronic](https://github.com/aptible/supercronic)) — one container handles both the backup and the schedule.

```bash
cp .env.example .env       # then fill in values
docker compose up -d       # pulls ghcr.io/fathah/pg-backup-service:latest
```

By default it runs every day at 20:30 UTC (`CRON_SCHEDULE=30 20 * * *`). For setup details, scheduling syntax, host vs. container Postgres, and operations, see [docker.md](./docker.md).

---

## Development & Build from Source

### Prerequisites

- **Go**: 1.24 or higher.
- **PostgreSQL Client Tools**: The `pg_dumpall` utility must be installed.

### Local Build

```bash
go build -o pg-backup-service main.go
```

### CI/CD / GitHub Workflow

A workflow at [`.github/workflows/build.yml`](./.github/workflows/build.yml) runs on every push to the `release` branch:

- Builds a Linux binary and publishes it as a GitHub Release.
- Builds a multi-arch (amd64/arm64) Docker image and pushes it to `ghcr.io/fathah/pg-backup-service` tagged `latest`, `r<run_number>`, and `sha-<short-sha>`.

## Features

- **Database Discovery**: Automatically identifies all user-created databases on the server.
- **Granular Backups**: Per-database exports using `pg_dump` for easier restoration.
- **Global Data**: Captures roles, users, and groups using `pg_dumpall --globals-only`.
- **Hybrid Fallback**: Automatically falls back to a full `pg_dumpall` if database discovery fails.
- **Secure Upload**: Integrates with the ZDrive API using signed URLs for secure multipart transfers.
- **Configurable**: Managed entirely via environment variables.

## License

MIT [LICENSE](./LICENSE)

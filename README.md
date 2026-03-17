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

To schedule daily backups, refer to the [cronjob.md](./cronjob.md) file for setup instructions and a wrapper script.

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

A workflow is included in `.github/workflows/build.yml` that builds a Linux binary on every push to the `main` branch.

## Features

- **Database Discovery**: Automatically identifies all user-created databases on the server.
- **Granular Backups**: Per-database exports using `pg_dump` for easier restoration.
- **Global Data**: Captures roles, users, and groups using `pg_dumpall --globals-only`.
- **Hybrid Fallback**: Automatically falls back to a full `pg_dumpall` if database discovery fails.
- **Secure Upload**: Integrates with the ZDrive API using signed URLs for secure multipart transfers.
- **Configurable**: Managed entirely via environment variables.

## License

MIT [LICENSE](./LICENSE)

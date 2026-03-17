# PostgreSQL Backup Service

A Go-based service designed to automate full PostgreSQL database dumps and securely upload them to [ZDrive](https://ziqx.cc/drive).

## Features

- **Database Discovery**: Automatically identifies all user-created databases on the server.
- **Granular Backups**: Per-database exports using `pg_dump` for easier restoration.
- **Global Data**: Captures roles, users, and groups using `pg_dumpall --globals-only`.
- **Hybrid Fallback**: Automatically falls back to a full `pg_dumpall` if database discovery fails.
- **Secure Upload**: Integrates with the ZDrive API using signed URLs for secure multipart transfers.
- **Configurable**: Managed entirely via environment variables.
- **Automated Builds**: Includes a GitHub Action for building Ubuntu-compatible binaries.
- **Cron Ready**: Designed for easy integration with standard cron jobs.

## Prerequisites

- **Go**: 1.24 or higher (for building).
- **PostgreSQL Client Tools**: The `pg_dumpall` utility must be installed on the machine running the service.

## Configuration

The service is configured via environment variables. You can use a `.env` file for local development or set them in your deployment environment.

| Variable        | Description                    | Default     |
| --------------- | ------------------------------ | ----------- |
| `PG_HOST`       | PostgreSQL server host         | `localhost` |
| `PG_PORT`       | PostgreSQL server port         | `5432`      |
| `PG_USER`       | PostgreSQL user                | `postgres`  |
| `PG_PASSWORD`   | PostgreSQL password (Required) | -           |
| `ZDRIVE_KEY`    | Your ZDrive ID (Required)      | -           |
| `ZDRIVE_SECRET` | Your ZDrive secret (Required)  | -           |
| `BACKUP_PREFIX` | Prefix for the backup filename | `pg_backup` |

## Usage

### Local Build

```bash
go build -o pg-backup-service main.go
```

### Running the Service

Ensure your environment variables are set, then run:

```bash
./pg-backup-service
```

## Automation

### GitHub Workflow

A workflow is included in `.github/workflows/build.yml` that builds a Linux binary on every push to the `main` branch. The resulting binary is uploaded as a GitHub Action artifact.

### Cron Job

To schedule daily backups at 2 AM IST (8:30 PM UTC), refer to the [cronjob.md](./cronjob.md) file for setup instructions and a wrapper script.

## License

MIT

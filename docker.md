# Running pg-backup-service with Docker

The Docker image bundles the Go binary, the PostgreSQL 16 client tools (`pg_dump`, `pg_dumpall`, `psql`), and a built-in cron scheduler ([supercronic](https://github.com/aptible/supercronic)). One container handles both the backup and the schedule — no host crontab needed.

A pre-built multi-arch image (`linux/amd64`, `linux/arm64`) is published to GitHub Container Registry on every push to the `release` branch:

```
ghcr.io/fathah/pg-backup-service:latest
ghcr.io/fathah/pg-backup-service:r<run_number>
ghcr.io/fathah/pg-backup-service:sha-<short-sha>
```

## Files

- `docker-compose.yml` — pulls the pre-built image and reads config from `.env`.
- `.env.example` — template for `.env`. Copy and fill in.
- `Dockerfile` — used by CI to build the published image. Only needed if you want to build locally.
- `docker-entrypoint.sh` — picks scheduled mode vs one-shot mode based on `CRON_SCHEDULE`.

## Modes

The container has two modes, decided at startup by the `CRON_SCHEDULE` environment variable:

| `CRON_SCHEDULE` | Behavior |
| --- | --- |
| set (e.g. `30 20 * * *`) | Long-running. Supercronic runs the backup on the given schedule. |
| empty / unset | One-shot. Runs the backup once and exits. |

Use scheduled mode for normal operation. Use one-shot mode for ad-hoc runs (`docker compose run --rm ...`) or when you already have an external scheduler (Kubernetes CronJob, host cron, etc.).

## Setup

### 1. Create `.env`

Copy the template and fill in your values:

```bash
cp .env.example .env
```

`.env.example` documents every variable. Required:

- `PG_PASSWORD`
- `ZDRIVE_KEY`
- `ZDRIVE_SECRET`

Sensible defaults are documented for the rest. **Never commit `.env`** — it's in `.dockerignore` and should be in `.gitignore`.

> Note: `env_file` reads literal `KEY=VALUE` lines — no shell substitution and no implicit defaults. If you omit a variable from `.env`, it is unset in the container, not auto-defaulted. Keep `.env` aligned with `.env.example`.

### 2. Start

```bash
docker compose up -d
```

Compose pulls `ghcr.io/fathah/pg-backup-service:latest` and starts the container in scheduled mode. Logs (including supercronic's per-run output) go to stdout:

```bash
docker compose logs -f pg-backup
```

To pull a newer published image later:

```bash
docker compose pull && docker compose up -d
```

### 3. Run an ad-hoc backup

To run once outside the schedule (e.g. to verify config), override `CRON_SCHEDULE` to empty:

```bash
docker compose run --rm -e CRON_SCHEDULE= pg-backup
```

The container will execute the backup, print logs, and exit.

## Connecting to Postgres

### Postgres on the host

On macOS / Windows Docker Desktop, set in `.env`:

```env
PG_HOST=host.docker.internal
```

On Linux, either use the host's LAN IP, or add `extra_hosts` to the service in `docker-compose.yml`:

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

### Postgres in another container

If Postgres runs in a sibling Compose service, put both on the same network and use the service name as `PG_HOST` (e.g. `PG_HOST=postgres`).

### Postgres on a remote server

Just set `PG_HOST` to the hostname/IP. Make sure the DB allows connections from the Docker host.

## Cron schedule reference

Supercronic uses standard 5-field cron syntax, evaluated in the container's `TZ`:

```
┌───────── minute (0-59)
│ ┌─────── hour (0-23)
│ │ ┌───── day of month (1-31)
│ │ │ ┌─── month (1-12)
│ │ │ │ ┌─ day of week (0-6, Sunday=0)
│ │ │ │ │
30 20 * * *   # every day at 20:30
0 2 * * *     # every day at 02:00
0 */6 * * *   # every 6 hours
0 3 * * 0     # Sundays at 03:00
```

Switch timezone by setting `TZ` (e.g. `TZ=Asia/Kolkata`). Both supercronic and your logs will follow it.

## Operations

| Action | Command |
| --- | --- |
| Start (pulls image if missing) | `docker compose up -d` |
| Pull a newer published image | `docker compose pull && docker compose up -d` |
| Stop | `docker compose down` |
| Tail logs | `docker compose logs -f pg-backup` |
| Run once now | `docker compose run --rm -e CRON_SCHEDULE= pg-backup` |
| Open a shell in the running container | `docker compose exec pg-backup bash` |
| Pin to a specific version | edit `image:` in `docker-compose.yml` to `ghcr.io/fathah/pg-backup-service:r42` (or a `sha-...` tag) |

## Building locally (optional)

You normally don't need to build the image yourself — CI publishes it. But if you've modified the source and want to test before pushing:

```bash
docker build -t pg-backup-service:dev .
```

Then temporarily change `image:` in `docker-compose.yml` to `pg-backup-service:dev` and run `docker compose up -d`.

## Troubleshooting

- **`pg_dump: server version mismatch`** — the published image ships `postgresql16-client`. If your server is a different major version, you'll need to build a custom image: edit `postgresql16-client` in the `Dockerfile` (e.g. `postgresql15-client`), rebuild locally, and point `image:` at your custom tag. The client major must be ≥ the server major.
- **`connection refused` to Postgres** — `PG_HOST=localhost` from inside a container points at the container itself, not the host. See the "Connecting to Postgres" section above.
- **Cron isn't firing** — check `docker compose logs pg-backup`; supercronic logs `schedule` and `next run` lines on startup. Verify `CRON_SCHEDULE` is set (it's empty in one-shot mode by design) and that `TZ` matches what you expect.
- **Container exits immediately** — that's one-shot mode. Either `CRON_SCHEDULE` is unset (or blank in `.env`), or the backup hit a fatal error. Check the logs.
- **`unauthorized` when pulling** — if your fork's GHCR package is private, log in first: `echo $GITHUB_TOKEN | docker login ghcr.io -u <your-username> --password-stdin`. Or make the package public in your repo's Packages settings.

## Security notes

- The container runs as a non-root `app` user.
- Secrets are passed via `.env`. For production, prefer Docker secrets, a secret manager, or your orchestrator's native mechanism — don't bake `.env` into images or commit it.
- The image has no inbound ports; nothing is exposed.

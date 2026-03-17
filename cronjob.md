# Cron Job Setup for pg-backup-service

To run the backup service every day at 8:30 PM UTC, follow these steps:

### 1. Preparation
Ensure the binary is built and you have a `.env` file in the same directory.

### 2. Create a Run Script (Optional but Recommended)
Create a file named `run_backup.sh` in your backup service directory:

```bash
#!/bin/bash
cd /path/to/pg-backup-service
set -a
source .env
set +a
./pg-backup-service
```
Make it executable: `chmod +x run_backup.sh`

### 3. Add to Crontab
Open your crontab editor: `crontab -e`

Add the following line (assuming your server uses UTC time):
```cron
30 20 * * * /path/to/pg-backup-service/run_backup.sh >> /path/to/pg-backup-service/backup.log 2>&1
```

If your server already uses Indian Standard Time (IST):
```cron
0 2 * * * /path/to/pg-backup-service/run_backup.sh >> /path/to/pg-backup-service/backup.log 2>&1
```

### Note on Environment Variables
The cron job environment is minimal. Ensure the script above correctly sets all required environment variables (`PG_PASSWORD`, `ZDRIVE_KEY`, `ZDRIVE_SECRET`, etc.).

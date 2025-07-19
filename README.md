# Docker Volume Google Drive Backup
Inspired by https://github.com/offen/docker-volume-backup

Docker Hub - [luizguilhermesj/docker-volume-google-drive-backup](https://hub.docker.com/r/luizguilhermesj/docker-volume-google-drive-backup)

A Docker container that compresses folders from `/backup/` and uploads them to Google Drive with retention policy support.

## Features

- Compresses each folder into a tar.gz archive
- Uploads to Google Drive
- Supports Google Workspace domain-wide delegation (impersonation)
- Configurable retention policy to delete old backups
- Timezone support via TZ environment variable
- RFC3339 timestamp format with optional filename-safe mode

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOOGLE_CREDENTIALS` | Path to Google service account credentials JSON | `/creds/credentials.json` |
| `GDRIVE_FOLDER_ID` | Google Drive folder ID or shared drive ID to upload to | `root` |
| `GDRIVE_IMPERSONATE_SUBJECT` | Email to impersonate (for domain-wide delegation) | (none) |
| `RETENTION_DAYS` | Number of days to keep backups (0 = delete all) | `30` |
| `TZ` | Timezone for timestamps (e.g., `America/New_York`) | (system default) |
| `FILENAME_SAFE_TIMESTAMP` | Set to `true` to replace colons with dashes in timestamps | `false` |
| `DEBUG` | Set to any value to enable debug logging | (disabled) |

## Timestamp Format

The script uses RFC3339 format for timestamps by default:
- **Default format**: `2025-07-19T21:10:56-03:00`
- **Filename-safe format** (when `FILENAME_SAFE_TIMESTAMP=true`): `2025-07-19T21-10-56-03-00`

The filename-safe format replaces colons with dashes to ensure compatibility with all filesystems, including Windows.



## Docker Compose Example

```yaml
version: '3.8'
services:
  backup:
    build: .
    volumes:
      - ./backup:/backup
      - ./credentials.json:/creds/credentials.json
    environment:
      - GOOGLE_CREDENTIALS=/creds/credentials.json # default
      - GDRIVE_FOLDER_ID=your-folder-id
      - GDRIVE_IMPERSONATE_SUBJECT=user@yourdomain.com
      - RETENTION_DAYS=30 # default
      - TZ=America/New_York # default to host
      - FILENAME_SAFE_TIMESTAMP=false #default
      - DEBUG=false # default
```

## Usage

1. Place your Google service account credentials in `./creds/`
2. Put folders to backup in `./backup/`
3. Set environment variables as needed
4. Run the container

The script will:
1. Set the timezone if `TZ` is specified
2. Clean up old backups based on retention policy
3. Compress each folder in `/backup/` to a tar.gz file
4. Upload each archive to Google Drive
5. Delete the local archive after successful upload

## Google Drive Setup

1. Create a service account in Google Cloud Console
2. Enable the Google Drive API
3. Download the service account JSON credentials
4. For domain-wide delegation, add the service account client ID to your Google Workspace admin console
5. Share the target folder with the service account email (or use impersonation) 
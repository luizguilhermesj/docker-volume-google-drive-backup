# Docker Volume Google Drive Backup
Inspired by https://github.com/offen/docker-volume-backup

Docker Hub - [luizguilhermesj/docker-volume-google-drive-backup](https://hub.docker.com/r/luizguilhermesj/docker-volume-google-drive-backup)  
Github - [github.com/luizguilhermesj/docker-volume-google-drive-backup](https://github.com/luizguilhermesj/docker-volume-google-drive-backup)

A Docker container that compresses folders from `/backup/` and uploads them to Google Drive with retention policy support.

## Features

- Compresses each folder into a tar.gz archive
- Uploads to Google Drive
- Supports Google Workspace domain-wide delegation (impersonation)
- Configurable retention policy to delete old backups
- Timezone support via TZ environment variable
- RFC3339 timestamp format with optional filename-safe mode
- Configurable chunk size for resumable uploads
- **Minimal Docker image** (~10MB) built with multi-stage build
- **Go implementation** for better performance and smaller footprint

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOOGLE_CREDENTIALS` | Path to Google service account credentials JSON | `/creds/credentials.json` |
| `GDRIVE_FOLDER_ID` | Google Drive folder ID or shared drive ID to upload to | `root` |
| `GDRIVE_IMPERSONATE_SUBJECT` | Email to impersonate (for domain-wide delegation) | (none) |
| `RETENTION_DAYS` | Number of days to keep backups (0 = delete all) | `30` |
| `TZ` | Timezone for timestamps (e.g., `America/New_York`) | (system default) |
| `FILENAME_SAFE_TIMESTAMP` | Set to `true` to replace colons with dashes in timestamps | `false` |
| `UPLOAD_CHUNK_SIZE` | Chunk size for resumable uploads (e.g., `1GB`, `8MB`, `16MB`) | (Google API default) |
| `UPLOAD_SPLIT_SIZE` | Split large files into chunks before uploading (e.g., `100MB`, `1GB`). If not set, no splitting occurs. | (disabled) |
| `DEBUG` | Set to any value to enable debug logging | (disabled) |

## Timestamp Format

The script uses RFC3339 format for timestamps by default:
- **Default format**: `2025-07-19T21:10:56-03:00`
- **Filename-safe format** (when `FILENAME_SAFE_TIMESTAMP=true`): `2025-07-19T21-10-56-03-00`

The filename-safe format replaces colons with dashes to ensure compatibility with all filesystems, including Windows.

## Upload Options

### Upload Behavior
The script uses Google Drive's Media API which automatically handles resumable uploads for all files. The chunk size configuration allows you to customize how the upload is performed.

### Upload Configuration

#### Chunk Size Configuration
- **Default behavior**: Uses Google Drive API's default chunk size (16MB)
- **Custom chunk size**: Set `UPLOAD_CHUNK_SIZE` to override the default
- Supports human-readable formats: `8MB`, `16MB`, `1GB`, `2GB`, etc.
- Supported units: B, KB, MB, GB, TB (case-insensitive)
- Common chunk sizes: `8MB`, `16MB`, `32MB`, `1GB`
- Smaller chunks = more API calls, may be more likely to hit rate limits, less likely to fail when connection is unstable
- Larger chunks = fewer API calls, potentially fewer rate limit issues, more likely to fail when connection is unstable

#### File Splitting
- **Split large files**: Files are only split if `UPLOAD_SPLIT_SIZE` is set and valid.
- If `UPLOAD_SPLIT_SIZE` is not set, no splitting occurs (files are uploaded whole, regardless of size).
- Files larger than the split size will be divided into chunks (e.g., `100MB`, `1GB`).
- Each chunk is uploaded as a separate file with `.part001`, `.part002`, etc. naming.
- Useful for very large files or to avoid rate limits by uploading smaller pieces.
- Chunk files are automatically cleaned up after successful upload.

### Rate Limiting
If you encounter HTTP 429 (Too Many Requests) errors:
- Try using bigger chunk sizes (e.g., `100MB` or `1GB`)
- Try using file splitting
- Consider running backups during off-peak hours
- Monitor your Google Drive API quota usage

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
      - GDRIVE_FOLDER_ID=your-folder-id
      - GDRIVE_IMPERSONATE_SUBJECT=user@yourdomain.com
      - RETENTION_DAYS=60
      - UPLOAD_CHUNK_SIZE=1GB # 1GB chunks (optional)
      - UPLOAD_SPLIT_SIZE=100MB # Split files larger than 100MB (optional)
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

# Skillcape Transcoder

A video transcoding service with REST API, async job processing, and Google Drive upload.

## Features

- **REST API** - Upload videos, track job progress, manage transcoding jobs
- **Async Processing** - Queue-based job system with configurable worker pool
- **FFmpeg Transcoding** - Converts videos to H.264/AAC MP4 format
- **Google Drive Upload** - Automatically uploads completed files to Google Drive
- **Webhook Notifications** - Receive callbacks when jobs complete
- **Persistent Jobs** - SQLite storage survives restarts
- **Docker Ready** - Multi-stage build with FFmpeg included

## Quick Start

### Using Docker (Recommended)

```bash
# Pull the image
docker pull ghcr.io/yourusername/skillcape-transcoder:latest

# Run with minimal config
docker run -d \
  -p 8080:8080 \
  -e API_KEY=your-secret-key \
  -v transcoder-data:/data \
  ghcr.io/yourusername/skillcape-transcoder:latest
```

### Using Docker Compose

1. Clone the repository
2. Create your configuration:
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```
3. (Optional) Add Google Drive credentials to `config/credentials.json`
4. Start the service:
   ```bash
   docker-compose up -d
   ```

## Configuration

All configuration is done via environment variables. Create a `.env` file or pass them directly to Docker.

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `API_KEY` | Secret key for API authentication. Clients must send this in the `X-API-Key` header. | `sk-abc123xyz` |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `WORKER_COUNT` | `2` | Number of concurrent transcoding workers |
| `TEMP_DIR` | `/tmp/transcoder` | Directory for uploads, outputs, and database |
| `WEBHOOK_URL` | *(none)* | URL to POST job completion notifications |
| `WEBHOOK_RETRY_COUNT` | `3` | Number of retry attempts for failed webhooks |

### Google Drive Variables

To enable automatic upload to Google Drive, configure these variables:

| Variable | Description |
|----------|-------------|
| `GOOGLE_CREDENTIALS_FILE` | Path to service account JSON file (default: `/config/credentials.json`) |
| `GOOGLE_DRIVE_FOLDER_ID` | ID of the destination folder in Google Drive |

## Google Drive Setup

1. **Create a Google Cloud Project**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select an existing one

2. **Enable the Google Drive API**
   - Navigate to "APIs & Services" → "Library"
   - Search for "Google Drive API" and enable it

3. **Create a Service Account**
   - Go to "APIs & Services" → "Credentials"
   - Click "Create Credentials" → "Service Account"
   - Give it a name and create
   - Click on the service account → "Keys" → "Add Key" → "Create new key" → JSON
   - Download the JSON file and save it as `config/credentials.json`

4. **Share the Drive Folder**
   - Create a folder in Google Drive (or use an existing one)
   - Right-click → "Share"
   - Share with the service account email (found in the JSON file as `client_email`)
   - Give it "Editor" access

5. **Get the Folder ID**
   - Open the folder in Google Drive
   - The URL will be: `https://drive.google.com/drive/folders/FOLDER_ID_HERE`
   - Copy the folder ID and set it as `GOOGLE_DRIVE_FOLDER_ID`

## API Usage

### Authentication

All API endpoints (except `/health`) require the `X-API-Key` header:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/jobs
```

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check (no auth) |
| `POST` | `/api/v1/jobs` | Upload video and create job |
| `GET` | `/api/v1/jobs` | List all jobs |
| `GET` | `/api/v1/jobs/:id` | Get job status |
| `DELETE` | `/api/v1/jobs/:id` | Cancel/delete job |

### Example: Upload a Video

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "X-API-Key: your-api-key" \
  -F "file=@/path/to/video.mov"
```

Response:
```json
{
  "job": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "pending",
    "progress": 0,
    "original_name": "video.mov",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### Example: Check Job Status

```bash
curl http://localhost:8080/api/v1/jobs/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-API-Key: your-api-key"
```

Response (completed):
```json
{
  "job": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "progress": 100,
    "drive_url": "https://drive.google.com/file/d/abc123/view",
    "original_name": "video.mov",
    "created_at": "2024-01-15T10:30:00Z",
    "completed_at": "2024-01-15T10:35:00Z"
  }
}
```

For complete API documentation, see [API.md](API.md).

## Webhook Notifications

When a job completes, a POST request is sent to your configured `WEBHOOK_URL`:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "drive_url": "https://drive.google.com/file/d/abc123/view",
  "drive_file_id": "abc123",
  "original_name": "video.mov",
  "completed_at": "2024-01-15T10:35:00Z"
}
```

Failed jobs include an `error` field instead of `drive_url`.

## Development

### Prerequisites

- Go 1.22+
- FFmpeg installed and in PATH

### Run Locally

```bash
# Install dependencies
go mod tidy

# Set environment variables
export API_KEY=dev-key
export TEMP_DIR=./data

# Run
go run ./cmd/server
```

### Build

```bash
go build -o server ./cmd/server
```

### Build Docker Image

```bash
docker build -t skillcape-transcoder .
```

## Architecture

```
┌─────────────┐     ┌──────────────────────────────────────────────────┐
│   Client    │────▶│              Transcoder API                      │
└─────────────┘     │  ┌─────────┐  ┌───────────┐  ┌──────────────┐   │
                    │  │   API   │─▶│ Job Queue │─▶│   Workers    │   │
                    │  │  (Gin)  │  │ (Channel) │  │  (FFmpeg)    │   │
                    │  └─────────┘  └───────────┘  └──────┬───────┘   │
                    │                                      │          │
                    │                              ┌───────▼────────┐ │
                    │                              │ Google Drive   │ │
                    │                              │    Upload      │ │
                    └──────────────────────────────┴────────────────┴─┘
                                                           │
                                                   ┌───────▼────────┐
                                                   │  Google Drive  │
                                                   └────────────────┘
```

## License

MIT

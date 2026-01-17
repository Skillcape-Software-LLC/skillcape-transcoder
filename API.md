# Skillcape Transcoder API Documentation

Base URL: `http://localhost:8080`

## Authentication

All `/api/v1/*` endpoints require authentication via the `X-API-Key` header.

```bash
curl -H "X-API-Key: your-api-key-here" http://localhost:8080/api/v1/jobs
```

### Authentication Errors

| Status Code | Response |
|-------------|----------|
| 401 | `{"error": "missing API key"}` |
| 401 | `{"error": "invalid API key"}` |

---

## Endpoints

### Health Check

Check if the service is running. No authentication required.

**Request**
```
GET /health
```

**Response** `200 OK`
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

### Create Job

Upload a video file to create a new transcoding job.

**Request**
```
POST /api/v1/jobs
Content-Type: multipart/form-data
X-API-Key: your-api-key
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | file | Yes | Video file to transcode |

**Example**
```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "X-API-Key: your-api-key" \
  -F "file=@/path/to/video.mov"
```

**Response** `202 Accepted`
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

**Error Responses**

| Status | Response |
|--------|----------|
| 400 | `{"error": "no file uploaded"}` |
| 500 | `{"error": "failed to save uploaded file"}` |
| 500 | `{"error": "failed to create job"}` |
| 503 | `{"error": "job queue is full, please try again later"}` |

---

### Get Job

Retrieve the status of a specific job.

**Request**
```
GET /api/v1/jobs/:id
X-API-Key: your-api-key
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Job UUID |

**Example**
```bash
curl http://localhost:8080/api/v1/jobs/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-API-Key: your-api-key"
```

**Response** `200 OK`
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

**Error Responses**

| Status | Response |
|--------|----------|
| 404 | `{"error": "job not found"}` |

---

### List Jobs

Retrieve a paginated list of all jobs.

**Request**
```
GET /api/v1/jobs
X-API-Key: your-api-key
```

| Query Parameter | Type | Default | Description |
|-----------------|------|---------|-------------|
| `limit` | integer | 20 | Max results (1-100) |
| `offset` | integer | 0 | Number of results to skip |

**Example**
```bash
curl "http://localhost:8080/api/v1/jobs?limit=10&offset=0" \
  -H "X-API-Key: your-api-key"
```

**Response** `200 OK`
```json
{
  "jobs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "completed",
      "progress": 100,
      "drive_url": "https://drive.google.com/file/d/abc123/view",
      "original_name": "video.mov",
      "created_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T10:35:00Z"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "status": "processing",
      "progress": 45,
      "original_name": "another-video.mp4",
      "created_at": "2024-01-15T10:40:00Z"
    }
  ],
  "total": 25,
  "limit": 10,
  "offset": 0
}
```

**Error Responses**

| Status | Response |
|--------|----------|
| 500 | `{"error": "failed to list jobs"}` |

---

### Delete Job

Cancel a pending/processing job or delete a completed job.

**Request**
```
DELETE /api/v1/jobs/:id
X-API-Key: your-api-key
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Job UUID |

**Example**
```bash
curl -X DELETE http://localhost:8080/api/v1/jobs/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-API-Key: your-api-key"
```

**Response** `200 OK`
```json
{
  "message": "job deleted"
}
```

**Error Responses**

| Status | Response |
|--------|----------|
| 404 | `{"error": "job not found"}` |
| 500 | `{"error": "failed to delete job"}` |

---

## Data Schemas

### Job Object

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique job identifier (UUID) |
| `status` | string | Current job status |
| `progress` | integer | Transcoding progress (0-100) |
| `drive_url` | string | Google Drive shareable link (when completed) |
| `error` | string | Error message (when failed) |
| `original_name` | string | Original uploaded filename |
| `created_at` | string | ISO 8601 timestamp |
| `completed_at` | string | ISO 8601 timestamp (when finished) |

### Job Status Values

| Status | Description |
|--------|-------------|
| `pending` | Job queued, waiting for worker |
| `processing` | Currently transcoding |
| `completed` | Successfully finished and uploaded |
| `failed` | Transcoding or upload failed |
| `cancelled` | Job was cancelled by user |

---

## Webhook Payload

When a job completes (success or failure), a POST request is sent to the configured webhook URL.

**Request**
```
POST {WEBHOOK_URL}
Content-Type: application/json
User-Agent: Skillcape-Transcoder/1.0
```

**Success Payload**
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

**Failure Payload**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "failed",
  "error": "transcoding failed: ffmpeg exited with code 1",
  "original_name": "video.mov",
  "completed_at": "2024-01-15T10:35:00Z"
}
```

---

## Example Workflow

```bash
# 1. Upload a video
JOB_ID=$(curl -s -X POST http://localhost:8080/api/v1/jobs \
  -H "X-API-Key: your-api-key" \
  -F "file=@video.mov" | jq -r '.job.id')

echo "Job created: $JOB_ID"

# 2. Poll for status
while true; do
  STATUS=$(curl -s http://localhost:8080/api/v1/jobs/$JOB_ID \
    -H "X-API-Key: your-api-key")

  PROGRESS=$(echo $STATUS | jq -r '.job.progress')
  STATE=$(echo $STATUS | jq -r '.job.status')

  echo "Status: $STATE, Progress: $PROGRESS%"

  if [ "$STATE" = "completed" ] || [ "$STATE" = "failed" ]; then
    echo $STATUS | jq '.job'
    break
  fi

  sleep 5
done
```

---

## Rate Limits

No rate limits are enforced by default. Configure your reverse proxy (nginx, Caddy) for rate limiting in production.

## CORS

CORS is enabled for all origins (`*`). Allowed methods: `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`.

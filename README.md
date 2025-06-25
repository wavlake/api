# Wavlake API

A GCP Cloud Run HTTP server that authenticates users via Firebase Auth and Nostr's NIP-98 HTTP auth spec.

## Features

- NIP-98 authentication middleware
- Firebase Auth integration via Firestore
- Nostr track upload and management
- Audio compression with ffmpeg
- GCS integration for file storage
- Async processing via Cloud Functions
- Heartbeat endpoint with deployed commit SHA
- Cloud Run deployment ready

## Prerequisites

- Go 1.23+
- Google Cloud SDK
- Firebase project with Firestore enabled

## Local Development

1. Set environment variables:
```bash
export GOOGLE_CLOUD_PROJECT=your-project-id
export PORT=8080
```

2. Run the server:
```bash
go run cmd/server/main.go
```

## Firestore Setup

Create a `nostr_auth` collection with documents containing:
```json
{
  "pubkey": "hex_encoded_nostr_pubkey",
  "firebase_uid": "firebase_user_id",
  "active": true,
  "created_at": "2025-01-23T12:00:00Z",
  "last_used_at": "2025-01-23T12:00:00Z"
}
```

## Authentication

Requests must include a NIP-98 auth event in the Authorization header:

```
Authorization: Nostr base64_encoded_event
```

The event must be a kind 27235 event with:
- `u` tag: exact request URL
- `method` tag: HTTP method
- Valid signature
- Timestamp within 60 seconds

## Architecture Overview

### Infrastructure Components

1. **API Service** (Cloud Run)
   - Go backend handling authentication and track management
   - Manages track metadata in Firestore
   - Generates presigned URLs for direct GCS uploads
   - Processes audio files with ffmpeg

2. **Cloud Storage** (GCS)
   - Bucket: `wavlake-audio`
   - Stores original files in `tracks/original/`
   - Stores compressed files in `tracks/compressed/`
   - CORS configured for browser uploads

3. **Cloud Function** (`process-audio-upload`)
   - Triggered by GCS file uploads
   - Calls API webhook when files land in `tracks/original/`

4. **Firestore**
   - Stores user data and NostrTrack metadata
   - Collections: `users`, `nostr_tracks`

### Track Upload and Compression Flow

```
1. User Initiates Upload (Web Client)
   ↓
2. Client creates NIP-98 auth event
   ↓
3. POST /v1/tracks/nostr (API)
   - Validates NIP-98 auth
   - Creates NostrTrack in Firestore
   - Returns presigned URL for original file
   ↓
4. Client uploads file directly to GCS
   - Uses presigned URL
   - File lands in tracks/original/{uuid}.mp3
   ↓
5. GCS triggers Cloud Function (optional auto-processing)
   - Detects new file in tracks/original/
   - Can trigger basic validation
   ↓
6. User Requests Custom Compression (Web Client)
   - POST /v1/tracks/{id}/compress
   - Specifies desired formats, bitrates, quality levels
   - Can request multiple versions (e.g., 128k MP3, 256k AAC, 64k OGG)
   ↓
7. API processes compression requests
   - Downloads original from GCS
   - Uses ffmpeg with user-specified options
   - Uploads each compressed version to tracks/compressed/
   - Updates Firestore with compression metadata
   ↓
8. User Manages Visibility (Web Client)
   - PUT /v1/tracks/{id}/compression-visibility
   - Chooses which versions to make public
   - Controls what appears in Nostr events
   ↓
9. Client Gets Public Versions for Nostr
   - GET /v1/tracks/{id}/public-versions
   - Returns only user-selected public versions
   ↓
10. Client publishes Nostr event
    - Kind 31337 event with track info
    - Multiple file URLs with different qualities
    - Users can choose optimal version for their bandwidth
```

## Endpoints

### GET /heartbeat
Returns server status and deployed commit SHA. This endpoint does not require authentication.

### POST /v1/tracks/nostr
Create a new Nostr track upload. Requires NIP-98 authentication.

**Request:**
```json
{
  "title": "Track Title",
  "artist": "Artist Name",
  "description": "Optional description",
  "genre": "Electronic",
  "tags": ["tag1", "tag2"]
}
```

**Response:**
```json
{
  "id": "uuid",
  "presigned_url": "https://...",
  "original_url": "https://...",
  "compressed_url": "https://...",
  "status": "pending_upload"
}
```

### GET /v1/tracks/my
Get all tracks for the authenticated user. Requires NIP-98 authentication.

### GET /v1/tracks/:id
Get a specific track by ID. Public endpoint.

### DELETE /v1/tracks/:id
Delete a track. Requires NIP-98 authentication and ownership.

### POST /v1/tracks/webhook/process
Internal webhook endpoint called by Cloud Function to trigger audio processing.

### POST /v1/tracks/:id/compress
Request custom compression versions for a track. Requires NIP-98 authentication.

**Request:**
```json
{
  "compressions": [
    {
      "bitrate": 128,
      "format": "mp3",
      "quality": "medium",
      "sample_rate": 44100
    },
    {
      "bitrate": 256,
      "format": "aac",
      "quality": "high"
    },
    {
      "bitrate": 64,
      "format": "ogg",
      "quality": "low"
    }
  ]
}
```

### PUT /v1/tracks/:id/compression-visibility
Control which compression versions are public for Nostr event publishing.

**Request:**
```json
{
  "version_updates": [
    {
      "version_id": "version-uuid-1",
      "is_public": true
    },
    {
      "version_id": "version-uuid-2", 
      "is_public": false
    }
  ]
}
```

### GET /v1/tracks/:id/public-versions
Get public compression versions for generating Nostr kind 31337 events.

**Response:**
```json
{
  "success": true,
  "data": {
    "track_id": "uuid",
    "original_url": "https://...",
    "public_versions": [
      {
        "id": "version-uuid",
        "url": "https://...",
        "bitrate": 128,
        "format": "mp3",
        "quality": "medium",
        "sample_rate": 44100,
        "size": 5242880,
        "is_public": true
      }
    ]
  }
}
```

### POST /v1/auth/link-pubkey
Link a Nostr pubkey to a Firebase account. Requires both Firebase and NIP-98 authentication.

### GET /v1/auth/get-linked-pubkeys
Get all linked pubkeys for a Firebase user. Requires Firebase authentication.

### POST /v1/auth/unlink-pubkey
Unlink a Nostr pubkey from a Firebase account. Requires Firebase authentication.

## Deployment

The service can be deployed using Cloud Build:

```bash
gcloud builds submit --config cloudbuild.yaml
```

Make sure to:
1. Create a service account with necessary permissions
2. Enable required APIs (Cloud Run, Cloud Build, Firestore)
3. Configure IAM permissions
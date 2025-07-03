# Current Architecture: GCS-Based Audio Processing

## Overview

The Wavlake API is a Go-based REST service for audio track management that uses Google Cloud Storage (GCS) for all file operations. The system supports dual authentication (Firebase + Nostr NIP-98) and provides both new track uploads and legacy PostgreSQL data access.

## Storage: Google Cloud Storage (GCS)

The API uses GCS for all audio file storage and processing with a clean, organized path structure.

### Configuration
```bash
# Required Environment Variables
GOOGLE_CLOUD_PROJECT=wavlake-alpha
GCS_BUCKET_NAME=wavlake-audio
TEMP_DIR=/tmp

# Optional PostgreSQL (for legacy data)
PROD_POSTGRES_CONNECTION_STRING_RO=postgres://...
```

## Track Upload Flow

### 1. User Authentication
- **NIP-98 Authentication**: Cryptographic signature proving pubkey ownership
- **Firebase Integration**: Links Nostr pubkey to Firebase user account

### 2. API Generates Presigned GCS URL
```http
POST /v1/tracks/nostr
X-Nostr-Authorization: {nip98-signature}
Content-Type: application/json

{"extension": "mp3"}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "track-uuid",
    "presigned_url": "https://storage.googleapis.com/wavlake-audio/tracks/original/track-uuid.mp3?...",
    "original_url": "https://storage.googleapis.com/wavlake-audio/tracks/original/track-uuid.mp3"
  }
}
```

### 3. Client Uploads to GCS
- Direct PUT to GCS (no API involvement)
- Upload to: `gs://wavlake-audio/tracks/original/{track-id}.{ext}`
- Faster upload (direct to Google's infrastructure)

### 4. GCS Triggers Cloud Function
- Cloud Function: `process-audio-upload`
- Trigger: File created in `tracks/original/`
- Function calls API webhook: `POST /v1/tracks/webhook/process`

### 5. API Processes Audio
- Downloads original from GCS
- FFmpeg compression to 128kbps MP3
- Uploads compressed file to: `tracks/compressed/{track-id}.mp3`
- Updates Firestore with duration, size, URLs

## File Organization

```
gs://wavlake-audio/
├── tracks/
│   ├── original/                    # Original uploaded files
│   │   ├── {track-id}.mp3
│   │   ├── {track-id}.wav
│   │   └── {track-id}.flac
│   └── compressed/                  # Processed files
│       ├── {track-id}.mp3          # Default 128kbps
│       ├── {track-id}_v1.mp3       # Custom compression version 1
│       └── {track-id}_v2.aac       # Custom compression version 2
```

## Architecture Components

### 1. Storage Service
**File**: `internal/services/storage.go`
- Native GCS integration using Google Cloud Storage client
- Presigned URL generation for secure uploads
- Object metadata and lifecycle management

### 2. Audio Processing
**File**: `internal/utils/audio.go`
- FFmpeg-based audio processing
- Multiple format support (MP3, AAC, OGG)
- Quality validation and metadata extraction

### 3. Path Configuration
**File**: `internal/utils/storage_paths.go`
- Consistent GCS path structure
- Track ID extraction from paths
- Version management for compressed files

## Database Layer

### Primary Database: Firestore
Collections:
- **`nostr_tracks`**: Track metadata, URLs, processing status
- **`users`**: Firebase ↔ Nostr pubkey linking
- **`nostr_auth`**: Nostr authentication records

### Legacy Database: PostgreSQL (Read-Only)
- **Purpose**: Access legacy catalog API data
- **Connection**: Via Cloud SQL with VPC connector
- **Tables**: `users`, `artists`, `albums`, `tracks`, `user_pubkey`

## Key Files

### Core Implementation
- `cmd/server/main.go` - Application entry point and service initialization
- `internal/services/storage.go` - GCS storage service
- `internal/services/nostr_track.go` - Track management
- `internal/services/processing.go` - Audio processing logic
- `internal/handlers/tracks.go` - Track API endpoints
- `internal/handlers/legacy_handler.go` - Legacy PostgreSQL endpoints

### Configuration
- `cloudbuild.yaml` - Cloud Build deployment configuration
- `CLAUDE.md` - Development documentation and guidelines

## Infrastructure

### Google Cloud Resources
- **Cloud Run**: API service hosting
- **Cloud Storage**: Audio file storage (`wavlake-audio` bucket)
- **Cloud Functions**: `process-audio-upload` trigger
- **Firestore**: Primary database
- **Cloud SQL**: Legacy PostgreSQL database
- **VPC Connector**: Secure database access
- **Secret Manager**: Database connection strings, webhook secrets

### Environment Variables (Production)
```bash
GOOGLE_CLOUD_PROJECT=wavlake-alpha
GCS_BUCKET_NAME=wavlake-audio
TEMP_DIR=/tmp
PROD_POSTGRES_CONNECTION_STRING_RO=secret-managed
WEBHOOK_SECRET=secret-managed
```

## API Endpoints

### Track Management
- `POST /v1/tracks/nostr` - Create track and get presigned upload URL
- `GET /v1/tracks/my` - Get user's tracks
- `GET /v1/tracks/{id}` - Get specific track
- `DELETE /v1/tracks/{id}` - Soft delete track
- `POST /v1/tracks/webhook/process` - Processing webhook (Cloud Function → API)

### Legacy Data (PostgreSQL)
- `GET /v1/legacy/metadata` - Complete user metadata
- `GET /v1/legacy/tracks` - User's legacy tracks
- `GET /v1/legacy/artists` - User's legacy artists
- `GET /v1/legacy/albums` - User's legacy albums

### Authentication
- `POST /v1/auth/link-pubkey` - Link Nostr pubkey to Firebase account
- `POST /v1/auth/check-pubkey-link` - Check pubkey link status

## Deployment

```bash
# Deploy to Cloud Run
gcloud builds submit --config cloudbuild.yaml

# View logs
gcloud run services logs read api --region=us-central1

# Update environment variables
gcloud run services update api --region=us-central1 \
  --set-env-vars NEW_VAR=value
```

## Performance Characteristics

- **Upload Speed**: Direct to GCS (very fast)
- **Processing Time**: ~30 seconds for typical 3-minute MP3
- **Storage Cost**: GCS Standard pricing
- **Compute Cost**: Cloud Run pay-per-request + Cloud Functions
- **Database**: Firestore automatic scaling + Cloud SQL managed

## Content Moderation

Every track upload is associated with the uploader's Nostr pubkey, enabling:
- **User Identification**: Cryptographic pubkey tied to every file
- **Content Removal**: Query tracks by pubkey, delete via API
- **File Cleanup**: Remove files from GCS `tracks/original/` and `tracks/compressed/`

---

**Current Status**: Production-ready GCS implementation  
**Last Updated**: July 2025  
**API Version**: GCS-native architecture
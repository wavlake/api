# Current API Architecture (July 2025)

This document describes how the Wavlake API works **today** in production. This is the complete current state documentation covering storage, processing, authentication, and infrastructure.

## Architecture History

1. **Initial Implementation**: The API was originally built using Google Cloud Storage (GCS)
2. **S3 Migration (Current)**: Migrated to AWS S3 for compatibility with the legacy catalog API
3. **Future Plan**: Migrate back to GCS with enhanced features and modern architecture

## Storage: AWS S3

The API currently uses AWS S3 for all audio file storage and processing.

### Configuration
```bash
STORAGE_PROVIDER=s3
AWS_S3_BUCKET_NAME=wavlake-media          # Production
AWS_S3_BUCKET_NAME=staging-wavlake-media  # Staging
AWS_REGION=us-east-2
```

### File Structure
```
s3://wavlake-media/
├── raw/                    # Original uploaded files
│   └── {trackId}.{ext}    # Example: raw/abc-123-def.mp3
├── track/                  # Processed files (128kbps MP3)
│   └── {trackId}.mp3      # Example: track/abc-123-def.mp3
└── image/                  # Album artwork (future)
    └── {imageId}.{ext}    # Example: image/xyz-789-uvw.jpg
```

## How Track Upload Works

### 1. Client Requests Upload URL
```http
POST /v1/tracks/nostr
Content-Type: application/json
X-Nostr-Authorization: {nip98-signature}

{
  "extension": "mp3"
}
```

### 2. API Generates Presigned S3 URL
```json
{
  "success": true,
  "data": {
    "id": "abc-123-def",
    "presigned_url": "https://wavlake-media.s3.us-east-2.amazonaws.com/raw/abc-123-def.mp3?X-Amz-Algorithm=...",
    "original_url": "https://d1d8hh7a10hq2y.cloudfront.net/raw/abc-123-def.mp3",
    "compressed_url": null
  }
}
```

### 3. Client Uploads to S3
- Uses the presigned URL
- Direct PUT to S3 (no API involvement)
- File stored at `raw/abc-123-def.mp3`

### 4. S3 Triggers AWS Lambda
- Lambda function: `compressor-prod` or `compressor-staging`
- Detects new file in `raw/` prefix
- Calls API webhook

### 5. API Processes Audio
```http
POST /v1/tracks/webhook/process
Content-Type: application/json

{
  "track_id": "abc-123-def",
  "status": "uploaded",
  "source": "s3_trigger"
}
```

The API then:
1. Downloads original from `raw/abc-123-def.mp3`
2. Compresses to 128kbps MP3 using FFmpeg
3. Uploads compressed file to `track/abc-123-def.mp3`
4. Updates Firestore with metadata

## Database Schema

### Firestore Collection: `nostr_tracks`
```typescript
{
  id: "abc-123-def",                    // Track UUID
  user_id: "firebase-user-123",        // Owner's Firebase UID
  title: "My Track Title",             // Track title
  original_url: "https://d1d8hh7a10hq2y.cloudfront.net/raw/abc-123-def.mp3",
  compressed_url: "https://d1d8hh7a10hq2y.cloudfront.net/track/abc-123-def.mp3",
  is_processing: false,                // true during compression
  duration: 180,                       // Track length in seconds
  size: 4562944,                      // File size in bytes
  created_at: "2025-07-02T10:30:00Z",
  updated_at: "2025-07-02T10:32:15Z"
}
```

## Authentication

The API uses **NIP-98 (Nostr)** signatures for authentication:
- No traditional usernames/passwords
- Cryptographic signatures prove identity
- Decentralized - no central auth server

**Example Request:**
```http
POST /v1/tracks/nostr
X-Nostr-Authorization: Nostr eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

{
  "extension": "mp3"
}
```

## Code Architecture

### Key Components

1. **Storage Interface** (`internal/services/interfaces.go`)
   - Abstracts storage operations
   - Supports both S3 and GCS implementations
   - Enables future migration without changing business logic

2. **S3 Service** (`internal/services/storage_s3.go`)
   - AWS SDK v2 implementation
   - Handles presigned URLs for direct uploads
   - CloudFront CDN integration

3. **Path Configuration** (`internal/utils/storage_paths.go`)
   - Dynamic path generation based on storage provider
   - Maintains legacy compatibility for S3 (`raw/`, `track/`)
   - Supports modern paths for GCS (`tracks/original/`, `tracks/compressed/`)

4. **Processing Service** (`internal/services/processing.go`)
   - FFmpeg-based audio compression
   - Creates 128kbps MP3 files for all uploads
   - Updates Firestore with processing status

### Key Files
- `cmd/server/main.go` - Application entry point and service selection
- `internal/services/storage_s3.go` - S3 storage implementation
- `internal/services/interfaces.go` - Storage interface definition
- `internal/utils/storage_paths.go` - Path generation logic
- `internal/services/processing.go` - Audio processing
- `internal/services/nostr_track.go` - Track management

### Storage Interface
```go
type StorageServiceInterface interface {
    GeneratePresignedURL(ctx context.Context, objectName string, expiration time.Duration) (string, error)
    GetPublicURL(objectName string) string
    UploadObject(ctx context.Context, objectName string, data io.Reader, contentType string) error
    GetObjectMetadata(ctx context.Context, objectName string) (interface{}, error)
    GetObjectReader(ctx context.Context, objectName string) (io.ReadCloser, error)
}
```

This interface allows the API to switch storage providers without changing business logic.

## Infrastructure

### AWS Resources
- **S3 Buckets**: `wavlake-media`, `staging-wavlake-media`
- **Lambda Functions**: `compressor-prod`, `compressor-staging`
- **CloudFront CDN**: `d1d8hh7a10hq2y.cloudfront.net`
- **IAM User**: Service account for S3 access

### GCP Resources
- **Cloud Run**: Hosts the API service
- **Firestore**: Primary database
- **Secret Manager**: Stores AWS credentials
- **Cloud Build**: CI/CD pipeline
- **VPC Connector**: PostgreSQL legacy access

## Deployment

### Environment Variables
```bash
# Storage Configuration
STORAGE_PROVIDER=s3
AWS_REGION=us-east-2
AWS_S3_BUCKET_NAME=wavlake-media
AWS_S3_RAW_PREFIX=raw
AWS_S3_TRACK_PREFIX=track
AWS_S3_IMAGE_PREFIX=image

# GCP Configuration
GOOGLE_CLOUD_PROJECT=wavlake-alpha
GCS_BUCKET_NAME=wavlake-audio  # Not used when STORAGE_PROVIDER=s3

# Processing
TEMP_DIR=/tmp
```

### Secrets (in GCP Secret Manager)
- `aws-access-key-id` - AWS Access Key ID
- `AWS_S3_MEDIA_SECRET_PROD` - AWS Secret Access Key (production)
- `AWS_S3_MEDIA_SECRET_STAGING` - AWS Secret Access Key (staging)
- `webhook-secret` - API webhook authentication
- `PROD_POSTGRES_CONNECTION_STRING_RO` - Legacy PostgreSQL access

### Build & Deploy
```bash
# Deploy to staging with S3
gcloud builds submit --substitutions=_STORAGE_PROVIDER=s3,_S3_BUCKET=staging-wavlake-media,_AWS_SECRET=AWS_S3_MEDIA_SECRET_STAGING

# Deploy to production with S3
gcloud builds submit --substitutions=_STORAGE_PROVIDER=s3,_S3_BUCKET=wavlake-media,_AWS_SECRET=AWS_S3_MEDIA_SECRET_PROD
```

## Why We're Using S3 Today

The migration to S3 was necessary to:
1. **Maintain compatibility** with the existing catalog API
2. **Use existing infrastructure** (S3 buckets, Lambda functions)
3. **Minimize disruption** during the PostgreSQL-to-Firestore migration
4. **Leverage existing CDN** configuration

## Current Limitations

1. **Single Format**: Only produces 128kbps MP3 files
2. **Basic Processing**: No waveforms, previews, or multiple qualities
3. **Legacy Paths**: Constrained by catalog API compatibility (`raw/`, `track/`)
4. **Cross-Cloud**: S3 storage + GCP compute creates latency and costs
5. **Lambda Dependency**: Requires AWS Lambda for processing triggers

## Monitoring

### Health Check
```bash
curl https://api.wavlake.com/heartbeat
```

### Processing Status
Check Firestore `nostr_tracks` collection for `is_processing` field.

### S3 Contents
```bash
aws s3 ls s3://wavlake-media/raw/ --recursive | head -10
aws s3 ls s3://wavlake-media/track/ --recursive | head -10
```

## Performance

- **Upload Speed**: Direct to S3 (very fast)
- **Processing Time**: ~30-60 seconds for typical 3-5 minute track
- **CDN**: CloudFront provides global distribution
- **API Response**: ~200-500ms for metadata operations

## Next Steps

This is the **current production architecture**. For future improvements and migration plans, see:
- `S3_IMPLEMENTATION.md` - Technical details of the S3 implementation
- `GCS_MIGRATION_PLAN.md` - Detailed plan for future migration back to GCS

---

**Document Status**: Complete Current Production Architecture  
**Last Updated**: July 2025  
**API Version**: S3-based implementation
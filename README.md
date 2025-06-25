# Wavlake API

A GCP Cloud Run HTTP server that authenticates users via Firebase Auth and Nostr's NIP-98 HTTP auth spec.

## Features

- NIP-98 authentication middleware
- Firebase Auth integration via Firestore
- Nostr track upload and management
- Audio compression with ffmpeg (automatic 128k MP3 + optional custom formats)
- User-controlled compression with multiple formats (MP3, AAC, OGG)
- Compression quality and bitrate controls
- Public/private version management for Nostr publishing
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

#### **Basic Upload (Automatic Compression)**
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
5. GCS triggers Cloud Function
   - Detects new file in tracks/original/
   - Calls API webhook automatically
   ↓
6. API processes track automatically
   - Downloads original from GCS
   - Compresses to 128kbps MP3 (default)
   - Uploads to tracks/compressed/
   - Updates compressed_url field
   ↓
7. Client publishes Nostr event
   - Kind 31337 event with track info
   - Uses compressed_url for streaming
```

#### **Advanced Upload (Custom Compression) - Optional**
```
6. User Requests Custom Compression (Web Client) - OPTIONAL
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
8. User Manages Visibility (Web Client) - OPTIONAL
   - PUT /v1/tracks/{id}/compression-visibility
   - Chooses which versions to make public
   - Controls what appears in Nostr events
   ↓
9. Client Gets Public Versions for Nostr - OPTIONAL
   - GET /v1/tracks/{id}/public-versions
   - Returns only user-selected public versions
   ↓
10. Client publishes enhanced Nostr event - OPTIONAL
    - Kind 31337 event with track info
    - Multiple file URLs with different qualities
    - Users can choose optimal version for their bandwidth
```

## Compression Options

### **Automatic Compression (Default)**
Every uploaded track automatically gets compressed to a 128kbps MP3 version. This ensures:
- ✅ **Backwards Compatibility**: Existing client apps continue to work
- ✅ **Fast Streaming**: Optimized for web playback
- ✅ **Universal Support**: MP3 works everywhere
- ✅ **Immediate Availability**: No additional user action required

The `compressed_url` field in track responses always contains this default compressed version.

### **Custom Compression (Optional)**
Advanced users can request additional compression versions with specific parameters:

#### **Supported Formats**
- **MP3**: Universal compatibility, good compression
- **AAC**: Better quality at same bitrate, modern standard
- **OGG**: Open source, excellent compression

#### **Bitrate Options**
- **32-128 kbps**: Low bandwidth, smaller files
- **128-256 kbps**: Balanced quality and size  
- **256-320 kbps**: High quality, larger files

#### **Quality Levels**
- **Low**: Optimized for small file size
- **Medium**: Balanced quality/size (default)
- **High**: Optimized for audio quality

#### **Sample Rates**
- **22050 Hz**: Lower quality, smaller files
- **44100 Hz**: CD quality (default)
- **48000 Hz**: Professional standard
- **96000 Hz**: High-resolution audio

### **Public/Private Versions**
Users can control which compression versions appear in their Nostr events:
- **Private**: Available only to track owner
- **Public**: Included in kind 31337 Nostr events for streaming
- **Default**: The automatic 128k MP3 is always public

This allows users to offer multiple quality options (e.g., 64k for mobile, 256k for desktop) in a single Nostr event.

### **When to Use Each Approach**

#### **Use Automatic Compression When:**
- ✅ Building a simple music app
- ✅ Want immediate track availability
- ✅ Don't need multiple quality options
- ✅ Prioritizing development speed
- ✅ 128kbps MP3 quality is sufficient

#### **Use Custom Compression When:**
- 🎯 Need multiple quality options for different devices
- 🎯 Want to offer lossless or high-quality versions
- 🎯 Targeting audiophiles with high-end equipment
- 🎯 Need specific formats (AAC for Apple, OGG for open source)
- 🎯 Want to optimize for different bandwidth scenarios
- 🎯 Building advanced music distribution features

## Endpoints

### **Core Endpoints (Required)**

#### GET /heartbeat
Returns server status and deployed commit SHA. This endpoint does not require authentication.

#### POST /v1/tracks/nostr
Create a new Nostr track upload. Requires NIP-98 authentication.

**Request:**
```json
{
  "extension": "mp3"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "presigned_url": "https://...",
    "original_url": "https://...",
    "compressed_url": "", // Populated after automatic compression
    "is_processing": true,
    "is_compressed": false
  }
}
```

#### GET /v1/tracks/my
Get all tracks for the authenticated user. Requires NIP-98 authentication.

#### GET /v1/tracks/:id
Get a specific track by ID. Public endpoint. Returns basic track info including `compressed_url`.

#### DELETE /v1/tracks/:id
Delete a track. Requires NIP-98 authentication and ownership.

#### POST /v1/tracks/webhook/process
Internal webhook endpoint called by Cloud Function to trigger audio processing.

### **Authentication Endpoints**

#### POST /v1/auth/link-pubkey
Link a Nostr pubkey to a Firebase account. Requires both Firebase and NIP-98 authentication.

#### GET /v1/auth/get-linked-pubkeys
Get all linked pubkeys for a Firebase user. Requires Firebase authentication.

#### POST /v1/auth/unlink-pubkey
Unlink a Nostr pubkey from a Firebase account. Requires Firebase authentication.

### **Advanced Compression Endpoints (Optional)**

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
3. Configure IAM permissions# Test change

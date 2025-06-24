# Nostr Track Upload API

This API provides Nostr-authenticated track upload functionality using NIP-98 authentication, GCP Cloud Storage, and Firestore with automatic audio processing.

## Features

- **NIP-98 Authentication**: Secure Nostr-based authentication for track operations
- **GCP Cloud Storage**: Scalable file storage with presigned upload URLs
- **Firestore Database**: Document-based storage for track metadata
- **Automatic Audio Processing**: Background compression to 128kbps MP3 for streaming
- **Dual File Access**: Both original lossless and compressed lossy versions available
- **Status Tracking**: Real-time processing status updates
- **Soft Deletion**: Tracks can be marked as deleted without permanent removal

## Complete Client Flow

### 1. Track Upload Initiation
Client creates a new track upload session and receives both a presigned upload URL and track metadata.

### 2. File Upload
Client uploads their high-quality file (FLAC, WAV, etc.) directly to GCS using the presigned URL.

### 3. Automatic Processing (Event-Driven)
GCS automatically triggers processing when file upload completes:
- **Cloud Function** detects file upload via GCS event trigger
- **API Webhook** receives notification and starts background processing
- **Processing Service** downloads, validates, and compresses the audio
- **Status Update** provides both original and compressed file URLs

*Note: Processing typically completes within 1-2 minutes depending on file size*

### 4. Status Monitoring
Client can poll the track status to monitor processing progress.

### 5. File Access
Once processed, client receives URLs for both:
- **Original file**: High-quality lossless version for downloads
- **Compressed file**: Optimized streaming version

### 6. Nostr Event Creation
Client can now create their signed Nostr track event with the appropriate file URL(s) based on their needs.

## Environment Variables

Required environment variables:

```bash
GOOGLE_CLOUD_PROJECT=your-project-id
GCS_BUCKET_NAME=your-storage-bucket
PORT=8080
TEMP_DIR=/tmp
```

Optional:
```bash
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
FIREBASE_SERVICE_ACCOUNT_KEY=/path/to/firebase-key.json
GIN_MODE=release
```

## API Endpoints

### Track Upload
`POST /v1/tracks/nostr`

Creates a new track upload session with presigned URL.

**Authentication**: NIP-98 required
**Request Body**:
```json
{
  "extension": "mp3"
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "firebase_uid": "user-id",
    "pubkey": "nostr-pubkey",
    "original_url": "https://storage.googleapis.com/bucket/tracks/original/uuid.mp3",
    "presigned_url": "https://storage.googleapis.com/...",
    "extension": "mp3",
    "is_processing": true,
    "is_compressed": false,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### Get My Tracks
`GET /v1/tracks/my`

Returns all tracks for the authenticated user.

**Authentication**: NIP-98 required

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "original_url": "...",
      "compressed_url": "...",
      "duration": 180,
      "size": 5242880,
      "is_processing": false,
      "is_compressed": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Get Track (Public)
`GET /v1/tracks/:id`

Returns public track information. Full details if owned by authenticated user.

**Authentication**: Optional (NIP-98)

### Get Track Status
`GET /v1/tracks/:id/status`

Returns detailed processing status for track owner.

**Authentication**: NIP-98 required (must be track owner)

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "original_url": "https://storage.googleapis.com/bucket/tracks/original/uuid.flac",
    "compressed_url": "https://storage.googleapis.com/bucket/tracks/compressed/uuid.mp3",
    "is_processing": false,
    "is_compressed": true,
    "duration": 180,
    "size": 52428800,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:02:30Z"
  }
}
```

### Trigger Processing
`POST /v1/tracks/:id/process`

Manually triggers processing for a track (useful if automatic processing failed).

**Authentication**: NIP-98 required (must be track owner)

### Delete Track
`DELETE /v1/tracks/:id`

Soft deletes a track (marks as deleted).

**Authentication**: NIP-98 required (must be track owner)

### Processing Webhook
`POST /v1/tracks/webhook/process`

Webhook endpoint for file processing notifications.

**Request Body**:
```json
{
  "track_id": "uuid",
  "status": "processed", // or "failed"
  "size": 5242880,
  "duration": 180,
  "compressed_url": "https://...",
  "error": "error message if failed"
}
```

## Data Models

### NostrTrack
```go
type NostrTrack struct {
    ID               string    `firestore:"id"`
    FirebaseUID      string    `firestore:"firebase_uid"`
    Pubkey           string    `firestore:"pubkey"`
    OriginalURL      string    `firestore:"original_url"`
    CompressedURL    string    `firestore:"compressed_url,omitempty"`
    Extension        string    `firestore:"extension"`
    Size             int64     `firestore:"size,omitempty"`
    Duration         int       `firestore:"duration,omitempty"`
    IsProcessing     bool      `firestore:"is_processing"`
    IsCompressed     bool      `firestore:"is_compressed"`
    Deleted          bool      `firestore:"deleted"`
    NostrKind        int       `firestore:"nostr_kind,omitempty"`
    NostrDTag        string    `firestore:"nostr_d_tag,omitempty"`
    CreatedAt        time.Time `firestore:"created_at"`
    UpdatedAt        time.Time `firestore:"updated_at"`
}
```

## Audio Processing

The API includes audio processing utilities for:

- **Format Validation**: Supports MP3, WAV, FLAC, AAC, OGG, M4A, WMA, AIFF, AU
- **Metadata Extraction**: Duration, size, bitrate, sample rate, channels using ffprobe
- **Compression**: Converts to 128kbps MP3, 44.1kHz, stereo using ffmpeg
- **Download & Process**: Can download from URLs and compress in one step

### Prerequisites

The audio processing requires `ffmpeg` and `ffprobe` to be installed:

```bash
# Ubuntu/Debian
sudo apt-get install ffmpeg

# macOS
brew install ffmpeg

# Docker
FROM alpine:latest
RUN apk add --no-cache ffmpeg
```

## Storage Structure

Files are stored in GCS with the following structure:

```
bucket/
├── tracks/
│   ├── original/
│   │   ├── uuid1.mp3
│   │   ├── uuid2.wav
│   │   └── ...
│   └── compressed/
│       ├── uuid1.mp3
│       ├── uuid2.mp3
│       └── ...
```

## Authentication Flow

1. User generates NIP-98 event with:
   - `kind: 27235`
   - `u` tag: Full request URL
   - `method` tag: HTTP method
   - Valid timestamp (within 60 seconds)

2. User includes event in Authorization header:
   ```
   Authorization: Nostr <base64-encoded-event>
   ```

3. API validates event and checks pubkey is linked to Firebase user

4. Request proceeds with context containing `pubkey` and `firebase_uid`

## Error Handling

All endpoints return consistent error format:

```json
{
  "success": false,
  "error": "Error description"
}
```

Common HTTP status codes:
- `400`: Bad request (invalid input)
- `401`: Unauthorized (missing/invalid auth)
- `403`: Forbidden (not track owner)
- `404`: Not found
- `500`: Internal server error

## Development

### Running Locally

1. Set environment variables
2. Install dependencies: `go mod download`
3. Run: `go run cmd/server/main.go`

### Testing

The API includes unit tests for services and handlers:

```bash
go test ./...
```

### Building

```bash
go build -o server cmd/server/main.go
```

## GCS Event Triggers

The API includes automatic processing via GCS event triggers. See `GCS_TRIGGERS_SETUP.md` for detailed setup instructions.

**Quick Setup (Cloud Functions)**:
```bash
cd cloud-function/
export GOOGLE_CLOUD_PROJECT=your-project-id
export GCS_BUCKET_NAME=your-bucket-name  
export API_BASE_URL=https://your-api.com
./deploy.sh
```

This replaces polling with immediate event-driven processing when files are uploaded.

## Detailed Client Flow Example

Here's a complete example of the client-side process:

### Step 1: Create Track Upload Session
```javascript
// Client creates NIP-98 event and initiates upload
const response = await fetch('/v1/tracks/nostr', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Nostr ${base64Event}`
  },
  body: JSON.stringify({ extension: 'flac' })
});

const { data: track } = await response.json();
console.log('Track created:', track.id);
console.log('Upload to:', track.presigned_url);
console.log('Original will be at:', track.original_url);
```

### Step 2: Upload File
```javascript
// Upload the high-quality audio file
const fileInput = document.getElementById('audioFile');
const file = fileInput.files[0]; // User's FLAC/WAV file

await fetch(track.presigned_url, {
  method: 'PUT',
  headers: {
    'Content-Type': file.type
  },
  body: file
});

console.log('File uploaded successfully');
```

### Step 3: Monitor Processing Status
```javascript
// Poll for processing completion
const pollStatus = async () => {
  const response = await fetch(`/v1/tracks/${track.id}/status`, {
    headers: {
      'Authorization': `Nostr ${base64Event}`
    }
  });
  
  const { data: status } = await response.json();
  
  if (status.is_processing) {
    console.log('Still processing...');
    setTimeout(pollStatus, 5000); // Check again in 5 seconds
  } else if (status.is_compressed) {
    console.log('Processing complete!');
    console.log('Original file:', status.original_url);
    console.log('Compressed file:', status.compressed_url);
    console.log('Duration:', status.duration, 'seconds');
    
    // Now client can create their Nostr event
    createNostrTrackEvent(status);
  }
};

// Start polling after upload
setTimeout(pollStatus, 10000); // Give some time for processing to start
```

### Step 4: Create Nostr Track Event
```javascript
const createNostrTrackEvent = (trackData) => {
  // Client decides which URL(s) to include based on their needs
  const event = {
    kind: 1, // or whatever kind for music tracks
    content: "Check out my new track!",
    tags: [
      ['url', trackData.compressed_url], // For streaming
      ['alt_url', trackData.original_url], // For high-quality download
      ['duration', trackData.duration.toString()],
      ['file_type', 'audio/mpeg'], // compressed version
      ['alt_file_type', 'audio/flac'], // original version
    ],
    created_at: Math.floor(Date.now() / 1000),
    pubkey: userPubkey
  };
  
  // Client signs the event with their private key
  const signedEvent = signEvent(event, userPrivateKey);
  
  // Publish to Nostr relays
  publishToRelays(signedEvent);
};
```

## File URL Strategy

The API provides both file versions to give clients maximum flexibility:

**For Streaming/Playback:**
- Use `compressed_url` (128kbps MP3)
- Smaller files, faster loading
- Good for web players, mobile apps

**For Downloads/High Quality:**
- Use `original_url` (lossless FLAC/WAV)
- Preserve original quality
- Better for audiophiles, archival

**For Nostr Events:**
- Include both URLs with different tags
- Let users choose quality preference
- Support progressive enhancement (start with compressed, offer original)
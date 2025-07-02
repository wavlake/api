# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The Wavlake API is a Go-based REST service for audio track management with dual authentication (Firebase + Nostr NIP-98), Google Cloud integration, and audio processing capabilities. It serves as the backend for decentralized music applications while maintaining traditional web app compatibility.

## Development Commands

### Essential Commands
```bash
# Local development
go run cmd/server/main.go

# Build the binary
make build

# Run tests
make test
go test -v ./...

# Run specific test
go test -v ./internal/handlers -run TestCheckPubkeyLink

# Linting and formatting
make lint
make fmt
make vet

# Clean build artifacts
make clean
```

### Docker & Deployment
```bash
# Build Docker image
make docker-build

# Deploy to Cloud Run
make deploy

# Direct Cloud Build submission
gcloud builds submit --config cloudbuild.yaml
```

### Environment Setup
Required environment variables for local development:
```bash
export GOOGLE_CLOUD_PROJECT=your-project-id
export PORT=8080
export GCS_BUCKET_NAME=wavlake-audio
export TEMP_DIR=/tmp

# PostgreSQL Configuration (optional - for legacy data access)
export PROD_POSTGRES_CONNECTION_STRING_RO="postgres://user:pass@host:5432/dbname?sslmode=require"
export POSTGRES_MAX_CONNECTIONS=10
export POSTGRES_MAX_IDLE_CONNECTIONS=5
```

## Architecture Overview

### Authentication Architecture
The system implements **dual authentication** patterns:

- **Firebase Auth**: Traditional JWT-based authentication for web apps
- **NIP-98 Nostr Auth**: Cryptographic signature-based authentication for decentralized protocols
- **Dual Auth**: Some endpoints require both authentication methods simultaneously

#### Authentication Middleware Stack
Located in `internal/auth/`:
- `FirebaseMiddleware`: Validates Bearer tokens, sets `firebase_uid` in context
- `NIP98Middleware`: Two variants:
  - `SignatureValidationMiddleware()`: Signature validation only (fast)
  - `Middleware()`: Full validation including database lookup
- `DualAuthMiddleware`: Requires both Firebase JWT and NIP-98 signature

#### Authentication Patterns by Endpoint Type
```go
// Firebase only
authGroup.GET("/get-linked-pubkeys", firebaseMiddleware.Middleware(), handler)

// NIP-98 signature validation only (no database lookup)
authGroup.POST("/check-pubkey-link", nip98Middleware.SignatureValidationMiddleware(handler))

// Full NIP-98 with database lookup
authGroup.POST("/tracks/nostr", nip98Middleware.Middleware(), handler)

// NIP-98 with Firebase UID lookup (for legacy endpoints)
legacyGroup.GET("/metadata", nip98Middleware.Middleware(), handler)

// Dual authentication required
authGroup.POST("/link-pubkey", dualAuthMiddleware.Middleware(), handler)
```

### Service Layer Architecture
Services follow interface-driven design patterns defined in `internal/services/interfaces.go`:

- **UserService**: Manages Firebase ↔ Nostr pubkey linking with transactional consistency
- **NostrTrackService**: Handles audio track lifecycle, compression versions, and visibility
- **StorageService**: Abstracts Google Cloud Storage operations and presigned URLs
- **ProcessingService**: Orchestrates asynchronous audio processing with FFmpeg
- **PostgresService**: Provides read-only access to legacy PostgreSQL database for user metadata

### Data Model Patterns
- **Firestore Collections**: `users`, `nostr_auth`, `nostr_tracks`
- **PostgreSQL Legacy Tables**: `users`, `artists`, `albums`, `tracks` (read-only access)
- **Transactional Updates**: Ensures consistency between User and NostrAuth collections
- **Denormalization**: ActivePubkeys stored in User model for performance
- **Soft Deletes**: Tracks marked as deleted rather than physically removed

### Legacy Data Integration
- **PostgreSQL Connection**: Via VPC connector for secure private network access
- **Authentication Flow**: NIP-98 signature → linked Firebase UID → PostgreSQL queries
- **Graceful Handling**: Returns empty arrays for users with no legacy data
- **Connection Pooling**: Configurable max connections and idle connections

### Request/Response Flow
1. **Middleware Chain**: CORS → Auth → Handler
2. **Context Propagation**: Auth info set in Gin context by middleware
3. **Service Layer**: Business logic with interface injection
4. **Error Handling**: Consistent JSON error responses with HTTP status codes

## Key Development Patterns

### Adding New Endpoints
1. Define request/response structs in `internal/handlers/`
2. Choose appropriate authentication middleware
3. Create service methods with interface definitions
4. Add comprehensive tests with mocks
5. Update route configuration in `cmd/server/main.go`

### Testing Patterns
- Use testify/suite for organized test structure
- Mock services using interfaces defined in `internal/services/interfaces.go`
- Mock implementations in `internal/mocks/`
- Context-based auth testing using mock middleware

### Audio Processing
- **Automatic Compression**: Every upload gets 128kbps MP3 version
- **Custom Compression**: Optional multiple formats (MP3, AAC, OGG) with configurable quality
- **Asynchronous Processing**: Background goroutines with timeout handling
- **Temporary File Management**: Proper cleanup in `/tmp` directory

### External System Integration
- **Firestore**: Primary database with transaction support
- **Cloud Storage**: Presigned URLs for secure direct uploads to `tracks/original/` and `tracks/compressed/`
- **Cloud Functions**: `process-audio-upload` triggers on GCS file uploads
- **FFmpeg**: External dependency for audio format conversion

## Common Debugging

### CORS Issues
CORS configuration in `cmd/server/main.go` includes specific origins. When adding new frontend domains, update the `AllowOrigins` slice.

### Authentication Failures
- Check that `X-Nostr-Authorization` header is properly formatted for NIP-98
- Verify timestamp is within 60 seconds for NIP-98 events
- Ensure Firebase JWT token is valid and not expired
- Check that pubkey exists and is active in `nostr_auth` collection

### File Upload Issues
- Verify presigned URLs are generated correctly
- Check GCS bucket permissions and CORS configuration
- Ensure Cloud Function `process-audio-upload` is triggered
- Monitor processing status via `is_processing` field

### Database Consistency
- User and NostrAuth linking uses Firestore transactions
- ActivePubkeys array in User model must stay in sync
- Check transaction rollback on errors

### Legacy Endpoint Issues
- **PostgreSQL Connection**: Verify `PROD_POSTGRES_CONNECTION_STRING_RO` is set and VPC connector is configured
- **Empty Responses**: Legacy endpoints return empty arrays (not errors) when no data exists for a user
- **Authentication**: Legacy endpoints require NIP-98 signatures with linked Firebase UIDs
- **VPC Connectivity**: Ensure `cloud-sql-postgres` VPC connector allows private IP access to Cloud SQL

## Testing

### Integration Tests
```bash
# Run integration tests (requires Firestore emulator)
go test -v ./internal/services -run Integration

# Run specific handler tests
go test -v ./internal/handlers -run TestAuthHandlers
```

### Mock Generation
Mocks are manually created in `internal/mocks/` following the interface patterns. When adding new service methods, update corresponding mocks.

## Legacy API Endpoints

The API provides read-only access to legacy PostgreSQL data via NIP-98 authenticated endpoints:

### Available Endpoints
- `GET /v1/legacy/metadata` - Complete user metadata (user, artists, albums, tracks)
- `GET /v1/legacy/tracks` - User's tracks from legacy system
- `GET /v1/legacy/artists` - User's artists from legacy system
- `GET /v1/legacy/albums` - User's albums from legacy system
- `GET /v1/legacy/artists/:artist_id/tracks` - Tracks for specific artist
- `GET /v1/legacy/albums/:album_id/tracks` - Tracks for specific album

### Authentication
All legacy endpoints require:
1. **NIP-98 signature** in `X-Nostr-Authorization` header
2. **Linked Firebase UID** - the pubkey must be linked to a Firebase UID
3. **PostgreSQL access** - the Firebase UID must exist in the legacy database

### Response Format
See `LEGACY_API_TYPES.md` for complete TypeScript interfaces. All endpoints return:
- **200 OK** with data (or empty arrays if no data found)
- **401 Unauthorized** if NIP-98 signature is invalid
- **401 Unauthorized** if pubkey is not linked to Firebase UID

### Error Handling
Legacy endpoints use graceful error handling:
- Missing users return empty metadata response with `user: null`
- Missing tracks/artists/albums return empty arrays `[]`
- No 500 errors for missing data - always returns valid JSON structure

## Deployment

The application is designed for Cloud Run deployment with:
- Dockerfile for containerization
- Cloud Build configuration in `cloudbuild.yaml`
- Service account authentication for GCP services
- Health check endpoint at `/heartbeat`
- Graceful shutdown handling

### Build Arguments
The Docker build includes `COMMIT_SHA` for version tracking accessible via the heartbeat endpoint.
# Wavlake API

A GCP Cloud Run HTTP server that authenticates users via Firebase Auth and Nostr's NIP-98 HTTP auth spec.

## Features

- NIP-98 authentication middleware
- Firebase Auth integration via Firestore
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

## Endpoints

### GET /heartbeat
Returns server status and deployed commit SHA. This endpoint does not require authentication.

## Deployment

The service can be deployed using Cloud Build:

```bash
gcloud builds submit --config cloudbuild.yaml
```

Make sure to:
1. Create a service account with necessary permissions
2. Enable required APIs (Cloud Run, Cloud Build, Firestore)
3. Configure IAM permissions
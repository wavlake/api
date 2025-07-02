# Legacy API TypeScript Interfaces

This document contains TypeScript interfaces for the legacy API endpoints that provide access to PostgreSQL data.

## Base Types

```typescript
// Common types used across multiple endpoints
interface User {
  id: string;
  name: string;
  lightning_address: string;
  msat_balance: number;
  amp_msat: number;
  artwork_url: string;
  profile_url: string;
  is_locked: boolean;
  created_at: string;
  updated_at: string;
}

interface Artist {
  id: string;
  user_id: string;
  name: string;
  artwork_url: string;
  artist_url: string;
  bio: string;
  twitter: string;
  instagram: string;
  youtube: string;
  website: string;
  npub: string;
  verified: boolean;
  deleted: boolean;
  msat_total: number;
  created_at: string;
  updated_at: string;
}

interface Album {
  id: string;
  artist_id: string;
  title: string;
  artwork_url: string;
  description: string;
  genre_id: number;
  subgenre_id: number;
  is_draft: boolean;
  is_single: boolean;
  deleted: boolean;
  msat_total: number;
  is_feed_published: boolean;
  published_at: string;
  created_at: string;
  updated_at: string;
}

interface Track {
  id: string;
  artist_id: string;
  album_id: string;
  title: string;
  order: number;
  play_count: number;
  msat_total: number;
  live_url: string;
  raw_url: string;
  size: number;
  duration: number;
  is_processing: boolean;
  is_draft: boolean;
  is_explicit: boolean;
  compressor_error: boolean;
  deleted: boolean;
  lyrics: string;
  created_at: string;
  updated_at: string;
  published_at: string;
}
```

## Endpoint Response Types

### GET /v1/legacy/metadata

Returns complete user metadata including user profile, artists, albums, and tracks.

```typescript
interface LegacyMetadataResponse {
  user: User;
  artists: Artist[];
  albums: Album[];
  tracks: Track[];
}
```

### GET /v1/legacy/tracks

Returns all user tracks.

```typescript
interface LegacyTracksResponse {
  tracks: Track[];
}
```

### GET /v1/legacy/artists

Returns all user artists.

```typescript
interface LegacyArtistsResponse {
  artists: Artist[];
}
```

### GET /v1/legacy/albums

Returns all user albums.

```typescript
interface LegacyAlbumsResponse {
  albums: Album[];
}
```

### GET /v1/legacy/artists/{artist_id}/tracks

Returns tracks for a specific artist.

```typescript
interface LegacyArtistTracksResponse {
  tracks: Track[];
}
```

### GET /v1/legacy/albums/{album_id}/tracks

Returns tracks for a specific album.

```typescript
interface LegacyAlbumTracksResponse {
  tracks: Track[];
}
```

## Usage Examples

```typescript
// Fetch complete metadata
const response = await fetch('/v1/legacy/metadata', {
  headers: {
    'X-Nostr-Authorization': nip98SignatureHeader
  }
});
const data: LegacyMetadataResponse = await response.json();

// Fetch tracks for specific artist
const artistId = '513168a2-bea8-48ff-b79a-dd02b00a6541';
const tracksResponse = await fetch(`/v1/legacy/artists/${artistId}/tracks`, {
  headers: {
    'X-Nostr-Authorization': nip98SignatureHeader
  }
});
const tracksData: LegacyArtistTracksResponse = await tracksResponse.json();
```

## Authentication

All endpoints require NIP-98 (Nostr) authentication:

```typescript
headers: {
  'X-Nostr-Authorization': nip98SignatureHeader
}
```

The NIP-98 signature must be from a pubkey that is linked to a Firebase UID in the system. The middleware will:
1. Validate the NIP-98 signature
2. Look up the Firebase UID linked to the pubkey 
3. Use that Firebase UID to query the PostgreSQL legacy data

## Error Responses

When authentication fails or other errors occur:

```typescript
interface ErrorResponse {
  error: string;
}
```

Common error responses:
- `401 Unauthorized`: Invalid or missing NIP-98 signature
- `403 Forbidden`: Pubkey not linked to a Firebase UID
- `500 Internal Server Error`: Database connection issues
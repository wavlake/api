package services

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/wavlake/api/internal/models"
)

type PostgresService struct {
	db *sql.DB
}

// NewPostgresService creates a new PostgreSQL service instance
func NewPostgresService(db *sql.DB) *PostgresService {
	return &PostgresService{
		db: db,
	}
}

// GetUserByFirebaseUID retrieves a user by their Firebase UID
func (p *PostgresService) GetUserByFirebaseUID(ctx context.Context, firebaseUID string) (*models.LegacyUser, error) {
	query := `
		SELECT id, name, COALESCE(lightning_address, '') as lightning_address, 
		       COALESCE(msat_balance, 0) as msat_balance, COALESCE(amp_msat, 1000) as amp_msat,
		       COALESCE(artwork_url, '') as artwork_url, COALESCE(profile_url, '') as profile_url,
		       COALESCE(is_locked, false) as is_locked, created_at, updated_at
		FROM user 
		WHERE id = $1 AND NOT COALESCE(is_locked, false)
	`

	var user models.LegacyUser
	err := p.db.QueryRowContext(ctx, query, firebaseUID).Scan(
		&user.ID,
		&user.Name,
		&user.LightningAddress,
		&user.MSatBalance,
		&user.AmpMsat,
		&user.ArtworkURL,
		&user.ProfileURL,
		&user.IsLocked,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserTracks retrieves all tracks for a user by Firebase UID
func (p *PostgresService) GetUserTracks(ctx context.Context, firebaseUID string) ([]models.LegacyTrack, error) {
	query := `
		SELECT t.id, t.artist_id, t.album_id, t.title, t."order", 
		       COALESCE(t.play_count, 0) as play_count, COALESCE(t.msat_total, 0) as msat_total,
		       t.live_url, COALESCE(t.raw_url, '') as raw_url, 
		       COALESCE(t.size, 0) as size, COALESCE(t.duration, 0) as duration,
		       COALESCE(t.is_processing, false) as is_processing, COALESCE(t.is_draft, false) as is_draft,
		       COALESCE(t.is_explicit, false) as is_explicit, COALESCE(t.compressor_error, false) as compressor_error,
		       COALESCE(t.deleted, false) as deleted, COALESCE(t.lyrics, '') as lyrics,
		       t.created_at, t.updated_at, t.published_at
		FROM track t
		JOIN album al ON t.album_id = al.id
		JOIN artist ar ON al.artist_id = ar.id
		WHERE ar.user_id = $1 AND NOT COALESCE(t.deleted, false)
		ORDER BY t.created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, firebaseUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks: %w", err)
	}
	defer rows.Close()

	var tracks []models.LegacyTrack
	for rows.Next() {
		var track models.LegacyTrack
		err := rows.Scan(
			&track.ID,
			&track.ArtistID,
			&track.AlbumID,
			&track.Title,
			&track.Order,
			&track.PlayCount,
			&track.MSatTotal,
			&track.LiveURL,
			&track.RawURL,
			&track.Size,
			&track.Duration,
			&track.IsProcessing,
			&track.IsDraft,
			&track.IsExplicit,
			&track.CompressorError,
			&track.Deleted,
			&track.Lyrics,
			&track.CreatedAt,
			&track.UpdatedAt,
			&track.PublishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, track)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tracks: %w", err)
	}

	return tracks, nil
}

// GetUserArtists retrieves all artists for a user by Firebase UID
func (p *PostgresService) GetUserArtists(ctx context.Context, firebaseUID string) ([]models.LegacyArtist, error) {
	query := `
		SELECT id, user_id, name, COALESCE(artwork_url, '') as artwork_url,
		       artist_url, COALESCE(bio, '') as bio, COALESCE(twitter, '') as twitter,
		       COALESCE(instagram, '') as instagram, COALESCE(youtube, '') as youtube,
		       COALESCE(website, '') as website, COALESCE(npub, '') as npub,
		       COALESCE(verified, false) as verified, COALESCE(deleted, false) as deleted,
		       COALESCE(msat_total, 0) as msat_total, created_at, updated_at
		FROM artist 
		WHERE user_id = $1 AND NOT COALESCE(deleted, false)
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, firebaseUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query artists: %w", err)
	}
	defer rows.Close()

	var artists []models.LegacyArtist
	for rows.Next() {
		var artist models.LegacyArtist
		err := rows.Scan(
			&artist.ID,
			&artist.UserID,
			&artist.Name,
			&artist.ArtworkURL,
			&artist.ArtistURL,
			&artist.Bio,
			&artist.Twitter,
			&artist.Instagram,
			&artist.Youtube,
			&artist.Website,
			&artist.Npub,
			&artist.Verified,
			&artist.Deleted,
			&artist.MSatTotal,
			&artist.CreatedAt,
			&artist.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artist: %w", err)
		}
		artists = append(artists, artist)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate artists: %w", err)
	}

	return artists, nil
}

// GetUserAlbums retrieves all albums for a user by Firebase UID
func (p *PostgresService) GetUserAlbums(ctx context.Context, firebaseUID string) ([]models.LegacyAlbum, error) {
	query := `
		SELECT al.id, al.artist_id, al.title, COALESCE(al.artwork_url, '') as artwork_url,
		       COALESCE(al.description, '') as description, COALESCE(al.genre_id, 0) as genre_id,
		       COALESCE(al.subgenre_id, 0) as subgenre_id, COALESCE(al.is_draft, false) as is_draft,
		       COALESCE(al.is_single, false) as is_single, COALESCE(al.deleted, false) as deleted,
		       COALESCE(al.msat_total, 0) as msat_total, COALESCE(al.is_feed_published, true) as is_feed_published,
		       al.published_at, al.created_at, al.updated_at
		FROM album al
		JOIN artist ar ON al.artist_id = ar.id
		WHERE ar.user_id = $1 AND NOT COALESCE(al.deleted, false)
		ORDER BY al.created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, firebaseUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query albums: %w", err)
	}
	defer rows.Close()

	var albums []models.LegacyAlbum
	for rows.Next() {
		var album models.LegacyAlbum
		err := rows.Scan(
			&album.ID,
			&album.ArtistID,
			&album.Title,
			&album.ArtworkURL,
			&album.Description,
			&album.GenreID,
			&album.SubgenreID,
			&album.IsDraft,
			&album.IsSingle,
			&album.Deleted,
			&album.MSatTotal,
			&album.IsFeedPublished,
			&album.PublishedAt,
			&album.CreatedAt,
			&album.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan album: %w", err)
		}
		albums = append(albums, album)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate albums: %w", err)
	}

	return albums, nil
}

// GetTracksByArtist retrieves all tracks for a specific artist
func (p *PostgresService) GetTracksByArtist(ctx context.Context, artistID string) ([]models.LegacyTrack, error) {
	query := `
		SELECT t.id, t.artist_id, t.album_id, t.title, t."order", 
		       COALESCE(t.play_count, 0) as play_count, COALESCE(t.msat_total, 0) as msat_total,
		       t.live_url, COALESCE(t.raw_url, '') as raw_url, 
		       COALESCE(t.size, 0) as size, COALESCE(t.duration, 0) as duration,
		       COALESCE(t.is_processing, false) as is_processing, COALESCE(t.is_draft, false) as is_draft,
		       COALESCE(t.is_explicit, false) as is_explicit, COALESCE(t.compressor_error, false) as compressor_error,
		       COALESCE(t.deleted, false) as deleted, COALESCE(t.lyrics, '') as lyrics,
		       t.created_at, t.updated_at, t.published_at
		FROM track t
		WHERE t.artist_id = $1 AND NOT COALESCE(t.deleted, false)
		ORDER BY t."order", t.created_at
	`

	rows, err := p.db.QueryContext(ctx, query, artistID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks by artist: %w", err)
	}
	defer rows.Close()

	var tracks []models.LegacyTrack
	for rows.Next() {
		var track models.LegacyTrack
		err := rows.Scan(
			&track.ID,
			&track.ArtistID,
			&track.AlbumID,
			&track.Title,
			&track.Order,
			&track.PlayCount,
			&track.MSatTotal,
			&track.LiveURL,
			&track.RawURL,
			&track.Size,
			&track.Duration,
			&track.IsProcessing,
			&track.IsDraft,
			&track.IsExplicit,
			&track.CompressorError,
			&track.Deleted,
			&track.Lyrics,
			&track.CreatedAt,
			&track.UpdatedAt,
			&track.PublishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, track)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tracks: %w", err)
	}

	return tracks, nil
}

// GetTracksByAlbum retrieves all tracks for a specific album
func (p *PostgresService) GetTracksByAlbum(ctx context.Context, albumID string) ([]models.LegacyTrack, error) {
	query := `
		SELECT id, artist_id, album_id, title, "order", 
		       COALESCE(play_count, 0) as play_count, COALESCE(msat_total, 0) as msat_total,
		       live_url, COALESCE(raw_url, '') as raw_url, 
		       COALESCE(size, 0) as size, COALESCE(duration, 0) as duration,
		       COALESCE(is_processing, false) as is_processing, COALESCE(is_draft, false) as is_draft,
		       COALESCE(is_explicit, false) as is_explicit, COALESCE(compressor_error, false) as compressor_error,
		       COALESCE(deleted, false) as deleted, COALESCE(lyrics, '') as lyrics,
		       created_at, updated_at, published_at
		FROM track 
		WHERE album_id = $1 AND NOT COALESCE(deleted, false)
		ORDER BY "order", created_at
	`

	rows, err := p.db.QueryContext(ctx, query, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks by album: %w", err)
	}
	defer rows.Close()

	var tracks []models.LegacyTrack
	for rows.Next() {
		var track models.LegacyTrack
		err := rows.Scan(
			&track.ID,
			&track.ArtistID,
			&track.AlbumID,
			&track.Title,
			&track.Order,
			&track.PlayCount,
			&track.MSatTotal,
			&track.LiveURL,
			&track.RawURL,
			&track.Size,
			&track.Duration,
			&track.IsProcessing,
			&track.IsDraft,
			&track.IsExplicit,
			&track.CompressorError,
			&track.Deleted,
			&track.Lyrics,
			&track.CreatedAt,
			&track.UpdatedAt,
			&track.PublishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, track)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tracks: %w", err)
	}

	return tracks, nil
}

// Ensure PostgresService implements the interface
var _ PostgresServiceInterface = (*PostgresService)(nil)

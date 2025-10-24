package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"vinylhound/shared/models"
)

var (
	// ErrPlaylistNotFound is returned when a playlist cannot be located.
	ErrPlaylistNotFound = errors.New("playlist not found")
)

// InMemoryRepository stores playlists in-memory for demo purposes.
type InMemoryRepository struct {
	mu             sync.RWMutex
	playlists      map[int64]*models.Playlist
	nextPlaylistID int64
	nextSongID     int64
}

// NewInMemoryRepository seeds the repository with a couple of playlists.
func NewInMemoryRepository() *InMemoryRepository {
	repo := &InMemoryRepository{
		playlists:      make(map[int64]*models.Playlist),
		nextPlaylistID: 1,
		nextSongID:     1,
	}

	repo.seed()
	return repo
}

func (r *InMemoryRepository) seed() {
	now := time.Now().UTC()

	repoPlaylists := []*models.Playlist{
		{
			Title: "Morning Spins",
			Owner: "Avery",
			Songs: []models.PlaylistSong{
				{Title: "Sunrise Echoes", Artist: "Luna Rivers", Album: "First Light", LengthSeconds: 212, Genre: "Ambient"},
				{Title: "Golden Hour Groove", Artist: "The Vinyl Set", Album: "Daybreak", LengthSeconds: 248, Genre: "Indie"},
				{Title: "Coffeehouse Conversation", Artist: "Muted Tones", Album: "Acoustic Corners", LengthSeconds: 198, Genre: "Acoustic"},
			},
		},
		{
			Title: "Late Night Depths",
			Owner: "Jordan",
			Songs: []models.PlaylistSong{
				{Title: "Neon Reflections", Artist: "City Ghosts", Album: "After Dark", LengthSeconds: 265, Genre: "Synthwave"},
				{Title: "Echo Chamber", Artist: "Glass Waves", Album: "Night Paths", LengthSeconds: 233, Genre: "Electronic"},
				{Title: "Blue Midnight", Artist: "Ella Brooks", Album: "Skyline", LengthSeconds: 241, Genre: "Jazz"},
				{Title: "Starfield", Artist: "Atlas Drift", Album: "Orbitals", LengthSeconds: 279, Genre: "Ambient"},
			},
		},
	}

	for _, p := range repoPlaylists {
		_, _ = r.Create(context.Background(), p)
	}
	// Override created timestamps to the same seed moment for deterministic output.
	r.mu.Lock()
	for _, playlist := range r.playlists {
		playlist.CreatedAt = now
	}
	r.mu.Unlock()
}

// List returns every playlist.
func (r *InMemoryRepository) List(_ context.Context) ([]*models.Playlist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*models.Playlist, 0, len(r.playlists))
	for _, playlist := range r.playlists {
		result = append(result, clonePlaylist(playlist))
	}
	return result, nil
}

// Get returns a playlist by id.
func (r *InMemoryRepository) Get(_ context.Context, id int64) (*models.Playlist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	playlist, ok := r.playlists[id]
	if !ok {
		return nil, ErrPlaylistNotFound
	}
	return clonePlaylist(playlist), nil
}

// Create persists a playlist.
func (r *InMemoryRepository) Create(_ context.Context, playlist *models.Playlist) (*models.Playlist, error) {
	if playlist == nil {
		return nil, errors.New("playlist is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	playlist.ID = r.nextPlaylistID
	r.nextPlaylistID++
	playlist.CreatedAt = time.Now().UTC()
	assignSongIDs(playlist, &r.nextSongID)
	playlist.SongCount = len(playlist.Songs)

	r.playlists[playlist.ID] = clonePlaylist(playlist)

	return clonePlaylist(playlist), nil
}

// Update replaces an existing playlist.
func (r *InMemoryRepository) Update(_ context.Context, id int64, playlist *models.Playlist) (*models.Playlist, error) {
	if playlist == nil {
		return nil, errors.New("playlist is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.playlists[id]
	if !ok {
		return nil, ErrPlaylistNotFound
	}

	playlist.ID = id
	playlist.CreatedAt = existing.CreatedAt
	assignSongIDs(playlist, &r.nextSongID)
	playlist.SongCount = len(playlist.Songs)

	r.playlists[id] = clonePlaylist(playlist)

	return clonePlaylist(playlist), nil
}

// Delete removes a playlist by id.
func (r *InMemoryRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.playlists[id]; !ok {
		return ErrPlaylistNotFound
	}
	delete(r.playlists, id)
	return nil
}

func clonePlaylist(src *models.Playlist) *models.Playlist {
	if src == nil {
		return nil
	}
	clone := *src
	if len(src.Songs) > 0 {
		clone.Songs = make([]models.PlaylistSong, len(src.Songs))
		copy(clone.Songs, src.Songs)
	}
	return &clone
}

func assignSongIDs(playlist *models.Playlist, nextID *int64) {
	for i := range playlist.Songs {
		if playlist.Songs[i].ID == 0 {
			playlist.Songs[i].ID = *nextID
			*nextID++
		}
	}
}

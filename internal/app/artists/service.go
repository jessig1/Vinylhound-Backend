package artists

import (
	"context"
	"sort"
	"strings"

	"vinylhound/internal/store"
)

// Artist represents a musical artist in the catalogue.
type Artist struct {
	Name string `json:"name"`
}

// Filter narrows the list of returned artists.
type Filter struct {
	Name string
}

// AlbumLister exposes the album queries needed to derive artist data.
type AlbumLister interface {
	List(ctx context.Context, filter store.AlbumFilter) ([]store.Album, error)
}

// Service provides artist-centric operations.
type Service interface {
	List(ctx context.Context, filter Filter) ([]Artist, error)
}

type service struct {
	albums AlbumLister
}

// New constructs an artist Service backed by the supplied album lister.
func New(albums AlbumLister) Service {
	return &service{albums: albums}
}

func (s *service) List(ctx context.Context, filter Filter) ([]Artist, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	albums, err := s.albums.List(ctx, store.AlbumFilter{})
	if err != nil {
		return nil, err
	}

	var (
		artists []Artist
		seen    = make(map[string]struct{})
		target  = strings.ToLower(strings.TrimSpace(filter.Name))
	)

	for _, album := range albums {
		name := strings.TrimSpace(album.Artist)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		if target != "" && !strings.Contains(strings.ToLower(name), target) {
			continue
		}
		seen[name] = struct{}{}
		artists = append(artists, Artist{Name: name})
	}

	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})
	return artists, nil
}

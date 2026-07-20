package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"flex.local/server/internal/collection"
	"flex.local/server/internal/config"
	"flex.local/server/internal/library"
	"flex.local/server/internal/media"
	"flex.local/server/internal/playback"
	"flex.local/server/internal/scanmanager"
	"flex.local/server/internal/tag"
)

type fakeLibraryService struct{}
type fakeMediaService struct{ favorites []media.File }
type fakePlaybackService struct{}
type fakeScanService struct{}
type fakeTagService struct{}
type fakeCollectionService struct{}

func (fakeLibraryService) List(context.Context) ([]library.Library, error) { return nil, nil }
func (fakeLibraryService) Add(context.Context, string, string) (library.Library, error) {
	return library.Library{}, nil
}
func (fakeLibraryService) Update(context.Context, string, string, string) (library.Library, error) {
	return library.Library{}, nil
}
func (fakeLibraryService) Delete(context.Context, string) error             { return nil }
func (fakeMediaService) List(context.Context, string) ([]media.File, error) { return nil, nil }
func (service fakeMediaService) Favorites(context.Context) ([]media.File, error) {
	return service.favorites, nil
}
func (fakeMediaService) Get(context.Context, string) (media.File, error) {
	return media.File{}, media.ErrNotFound
}
func (fakeMediaService) Home(context.Context) (media.Home, error) { return media.Home{}, nil }
func (fakeMediaService) Search(context.Context, string) ([]media.SearchResult, error) {
	return nil, nil
}
func (fakeMediaService) Folders(context.Context, string) ([]media.FolderAssignment, error) {
	return nil, nil
}
func (fakeMediaService) UpdateMetadata(context.Context, string, media.MetadataInput) (media.File, error) {
	return media.File{}, nil
}
func (fakeScanService) Scan(context.Context, string) (media.ScanResult, error) {
	return media.ScanResult{}, nil
}
func (fakeScanService) Trigger(string)                   {}
func (fakeScanService) Status(string) scanmanager.Status { return scanmanager.Status{State: "idle"} }
func (fakeMediaService) Thumbnail(context.Context, string) (string, error) {
	return "", media.ErrNotFound
}
func (fakeMediaService) Transcode(context.Context, string) (string, error) {
	return "", media.ErrNotFound
}
func (fakePlaybackService) Get(context.Context, string) (playback.Progress, error) {
	return playback.Progress{}, nil
}
func (fakePlaybackService) Save(context.Context, string, int64, int64) (playback.Progress, error) {
	return playback.Progress{}, nil
}
func (fakeTagService) List(context.Context) ([]tag.Tag, error)               { return nil, nil }
func (fakeTagService) Assignments(context.Context) ([]tag.Assignment, error) { return nil, nil }
func (fakeTagService) Create(context.Context, string, string) (tag.Tag, error) {
	return tag.Tag{}, nil
}
func (fakeTagService) ListForMedia(context.Context, string) ([]tag.Tag, error) { return nil, nil }
func (fakeTagService) SetForMedia(context.Context, string, []string) ([]tag.Tag, error) {
	return nil, nil
}
func (fakeCollectionService) List(context.Context) ([]collection.Collection, error) { return nil, nil }
func (fakeCollectionService) Create(context.Context, string) (collection.Collection, error) {
	return collection.Collection{}, nil
}
func (fakeCollectionService) ListForMedia(context.Context, string) ([]collection.Collection, error) {
	return nil, nil
}
func (fakeCollectionService) SetForMedia(context.Context, string, []string) ([]collection.Collection, error) {
	return nil, nil
}
func (fakeCollectionService) MediaIDs(context.Context, string) ([]string, error) { return nil, nil }
func (fakeCollectionService) Update(context.Context, string, string) (collection.Collection, error) {
	return collection.Collection{}, nil
}
func (fakeCollectionService) Delete(context.Context, string) error              { return nil }
func (fakeCollectionService) RemoveMedia(context.Context, string, string) error { return nil }

func TestHealth(t *testing.T) {
	server := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeLibraryService{}, fakeMediaService{}, fakeScanService{}, fakePlaybackService{}, fakeTagService{}, fakeCollectionService{})
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	var body healthResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if body.Status != "ok" || body.Service != "flex" {
		t.Fatalf("unexpected response: %#v", body)
	}
}

func TestFavorites(t *testing.T) {
	service := fakeMediaService{favorites: []media.File{{
		ID: "media-1", LibraryID: "library-1", Filename: "video.mp4", Title: "Ma vidéo", Favorite: true,
	}}}
	server := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeLibraryService{}, service, fakeScanService{}, fakePlaybackService{}, fakeTagService{}, fakeCollectionService{})
	request := httptest.NewRequest(http.MethodGet, "/api/favorites", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	var body mediaListResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "media-1" || !body.Items[0].Favorite {
		t.Fatalf("unexpected favorites response: %#v", body)
	}
}

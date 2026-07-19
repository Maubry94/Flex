package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"flex.local/server/internal/config"
	"flex.local/server/internal/library"
	"flex.local/server/internal/media"
	"flex.local/server/internal/playback"
	"flex.local/server/internal/scanmanager"
)

type fakeLibraryService struct{}
type fakeMediaService struct{}
type fakePlaybackService struct{}
type fakeScanService struct{}

func (fakeLibraryService) List(context.Context) ([]library.Library, error) { return nil, nil }
func (fakeLibraryService) Add(context.Context, string, string) (library.Library, error) {
	return library.Library{}, nil
}
func (fakeLibraryService) Update(context.Context, string, string, string) (library.Library, error) {
	return library.Library{}, nil
}
func (fakeLibraryService) Delete(context.Context, string) error             { return nil }
func (fakeMediaService) List(context.Context, string) ([]media.File, error) { return nil, nil }
func (fakeMediaService) Get(context.Context, string) (media.File, error) {
	return media.File{}, media.ErrNotFound
}
func (fakeMediaService) Home(context.Context) (media.Home, error) { return media.Home{}, nil }
func (fakeMediaService) Search(context.Context, string) ([]media.SearchResult, error) {
	return nil, nil
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

func TestHealth(t *testing.T) {
	server := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeLibraryService{}, fakeMediaService{}, fakeScanService{}, fakePlaybackService{})
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

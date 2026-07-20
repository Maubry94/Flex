package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"flex.local/server/internal/auth"
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
type fakeAuthService struct {
	authenticated bool
	role          string
}

func (fakeAuthService) Configured(context.Context) (bool, error) { return true, nil }
func (fakeAuthService) Setup(context.Context, string, string) (auth.Session, error) {
	return auth.Session{}, nil
}
func (fakeAuthService) Login(context.Context, string, string) (auth.Session, error) {
	return auth.Session{}, nil
}

func (service fakeAuthService) Authenticate(context.Context, string) (auth.User, error) {
	if !service.authenticated {
		return auth.User{}, auth.ErrUnauthenticated
	}
	role := service.role
	if role == "" {
		role = "admin"
	}
	return auth.User{ID: "admin", Role: role, Active: true}, nil
}
func (fakeAuthService) Logout(context.Context, string) error { return nil }
func (fakeAuthService) ChangePassword(context.Context, string, string, string) (auth.Session, error) {
	return auth.Session{}, nil
}
func (fakeAuthService) UpdateProfile(context.Context, string, string) (auth.User, error) {
	return auth.User{}, nil
}
func (fakeAuthService) ListUsers(context.Context) ([]auth.User, error) { return nil, nil }
func (fakeAuthService) CreateUser(context.Context, string, string, string) (auth.User, error) {
	return auth.User{}, nil
}
func (fakeAuthService) UpdateUser(context.Context, string, auth.UserInput) (auth.User, error) {
	return auth.User{}, nil
}
func (fakeAuthService) ResetPassword(context.Context, string, string) error { return nil }
func (fakeAuthService) DeleteUser(context.Context, string) error            { return nil }

func (fakeLibraryService) List(context.Context) ([]library.Library, error) { return nil, nil }
func (fakeLibraryService) Add(context.Context, string, string) (library.Library, error) {
	return library.Library{}, nil
}
func (fakeLibraryService) Update(context.Context, string, string, string) (library.Library, error) {
	return library.Library{}, nil
}
func (fakeLibraryService) Delete(context.Context, string) error                     { return nil }
func (fakeMediaService) List(context.Context, string, string) ([]media.File, error) { return nil, nil }
func (service fakeMediaService) Favorites(context.Context, string) ([]media.File, error) {
	return service.favorites, nil
}
func (fakeMediaService) Get(context.Context, string, string) (media.File, error) {
	return media.File{}, media.ErrNotFound
}
func (fakeMediaService) Home(context.Context, string) (media.Home, error) { return media.Home{}, nil }
func (fakeMediaService) Search(context.Context, string, string) ([]media.SearchResult, error) {
	return nil, nil
}
func (fakeMediaService) Folders(context.Context, string) ([]media.FolderAssignment, error) {
	return nil, nil
}
func (fakeMediaService) UpdateMetadata(context.Context, string, string, media.MetadataInput) (media.File, error) {
	return media.File{}, nil
}
func (fakeMediaService) SetFavorite(context.Context, string, string, bool) (media.File, error) {
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
func (fakePlaybackService) Get(context.Context, string, string) (playback.Progress, error) {
	return playback.Progress{}, nil
}
func (fakePlaybackService) Save(context.Context, string, string, int64, int64) (playback.Progress, error) {
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
func (fakeCollectionService) List(context.Context, string) ([]collection.Collection, error) {
	return nil, nil
}
func (fakeCollectionService) Create(context.Context, string, string) (collection.Collection, error) {
	return collection.Collection{}, nil
}
func (fakeCollectionService) ListForMedia(context.Context, string, string) ([]collection.Collection, error) {
	return nil, nil
}
func (fakeCollectionService) SetForMedia(context.Context, string, string, []string) ([]collection.Collection, error) {
	return nil, nil
}
func (fakeCollectionService) MediaIDs(context.Context, string, string) ([]string, error) {
	return nil, nil
}
func (fakeCollectionService) Update(context.Context, string, string, string) (collection.Collection, error) {
	return collection.Collection{}, nil
}
func (fakeCollectionService) Delete(context.Context, string, string) error              { return nil }
func (fakeCollectionService) RemoveMedia(context.Context, string, string, string) error { return nil }

func TestHealth(t *testing.T) {
	server := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeAuthService{authenticated: true}, fakeLibraryService{}, fakeMediaService{}, fakeScanService{}, fakePlaybackService{}, fakeTagService{}, fakeCollectionService{})
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
	server := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeAuthService{authenticated: true}, fakeLibraryService{}, service, fakeScanService{}, fakePlaybackService{}, fakeTagService{}, fakeCollectionService{})
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

func TestProtectedAPIRequiresAuthentication(t *testing.T) {
	server := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeAuthService{}, fakeLibraryService{}, fakeMediaService{}, fakeScanService{}, fakePlaybackService{}, fakeTagService{}, fakeCollectionService{})
	request := httptest.NewRequest(http.MethodGet, "/api/libraries", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}

func TestStandardUserCannotCreateLibrary(t *testing.T) {
	server := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeAuthService{authenticated: true, role: "user"}, fakeLibraryService{}, fakeMediaService{}, fakeScanService{}, fakePlaybackService{}, fakeTagService{}, fakeCollectionService{})
	request := httptest.NewRequest(http.MethodPost, "/api/libraries", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", response.Code)
	}
}

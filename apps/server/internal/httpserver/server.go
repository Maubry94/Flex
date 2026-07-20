package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flex.local/server/internal/collection"
	"flex.local/server/internal/config"
	"flex.local/server/internal/library"
	"flex.local/server/internal/media"
	"flex.local/server/internal/playback"
	"flex.local/server/internal/scanmanager"
	"flex.local/server/internal/tag"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type libraryService interface {
	List(ctx context.Context) ([]library.Library, error)
	Add(ctx context.Context, name string, path string) (library.Library, error)
	Update(ctx context.Context, id string, name string, path string) (library.Library, error)
	Delete(ctx context.Context, id string) error
}

type mediaService interface {
	List(ctx context.Context, userID string, libraryID string) ([]media.File, error)
	Favorites(ctx context.Context, userID string) ([]media.File, error)
	Get(ctx context.Context, userID string, id string) (media.File, error)
	Home(ctx context.Context, userID string) (media.Home, error)
	Search(ctx context.Context, userID string, query string) ([]media.SearchResult, error)
	Folders(ctx context.Context, libraryID string) ([]media.FolderAssignment, error)
	UpdateMetadata(ctx context.Context, userID string, id string, input media.MetadataInput) (media.File, error)
	SetFavorite(ctx context.Context, userID string, id string, favorite bool) (media.File, error)
	Thumbnail(ctx context.Context, id string) (string, error)
	Transcode(ctx context.Context, id string) (string, error)
}

type scanService interface {
	Scan(ctx context.Context, libraryID string) (media.ScanResult, error)
	Trigger(libraryID string)
	Status(libraryID string) scanmanager.Status
}

type playbackService interface {
	Get(ctx context.Context, userID string, mediaID string) (playback.Progress, error)
	Save(ctx context.Context, userID string, mediaID string, positionMS int64, durationMS int64) (playback.Progress, error)
}

type tagService interface {
	List(ctx context.Context) ([]tag.Tag, error)
	Assignments(ctx context.Context) ([]tag.Assignment, error)
	Create(ctx context.Context, name string, color string) (tag.Tag, error)
	ListForMedia(ctx context.Context, mediaID string) ([]tag.Tag, error)
	SetForMedia(ctx context.Context, mediaID string, tagIDs []string) ([]tag.Tag, error)
}
type collectionService interface {
	List(context.Context, string) ([]collection.Collection, error)
	Create(context.Context, string, string) (collection.Collection, error)
	Update(context.Context, string, string, string) (collection.Collection, error)
	Delete(context.Context, string, string) error
	ListForMedia(context.Context, string, string) ([]collection.Collection, error)
	SetForMedia(context.Context, string, string, []string) ([]collection.Collection, error)
	MediaIDs(context.Context, string, string) ([]string, error)
	RemoveMedia(context.Context, string, string, string) error
}

func New(cfg config.Config, logger *slog.Logger, authentication authService, libraries libraryService, mediaFiles mediaService, scans scanService, progress playbackService, tags tagService, collections collectionService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler)
	mux.HandleFunc("GET /api/auth/status", authStatusHandler(authentication, logger))
	mux.HandleFunc("POST /api/auth/setup", setupHandler(authentication, logger))
	mux.HandleFunc("POST /api/auth/login", loginHandler(authentication, logger))
	mux.HandleFunc("POST /api/auth/logout", logoutHandler(authentication, logger))
	mux.HandleFunc("PUT /api/auth/password", changePasswordHandler(authentication, logger))
	mux.HandleFunc("PATCH /api/auth/profile", updateProfileHandler(authentication, logger))
	mux.HandleFunc("GET /api/users", requireAdmin(listUsersHandler(authentication, logger)))
	mux.HandleFunc("POST /api/users", requireAdmin(createUserHandler(authentication, logger)))
	mux.HandleFunc("PATCH /api/users/{userID}", requireAdmin(updateUserHandler(authentication, logger)))
	mux.HandleFunc("PUT /api/users/{userID}/password", requireAdmin(resetUserPasswordHandler(authentication, logger)))
	mux.HandleFunc("DELETE /api/users/{userID}", requireAdmin(deleteUserHandler(authentication, logger)))
	mux.HandleFunc("GET /api/libraries", listLibrariesHandler(libraries, logger))
	mux.HandleFunc("POST /api/libraries", requireAdmin(createLibraryHandler(libraries, scans, logger)))
	mux.HandleFunc("PATCH /api/libraries/{libraryID}", requireAdmin(updateLibraryHandler(libraries, logger)))
	mux.HandleFunc("DELETE /api/libraries/{libraryID}", requireAdmin(deleteLibraryHandler(libraries, logger)))
	mux.HandleFunc("GET /api/libraries/{libraryID}/scan", scanStatusHandler(scans))
	mux.HandleFunc("POST /api/libraries/{libraryID}/scan", requireAdmin(scanLibraryHandler(scans, logger)))
	mux.HandleFunc("GET /api/libraries/{libraryID}/folders", libraryFoldersHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media", listMediaHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/favorites", favoritesHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/search", searchMediaHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}", getMediaHandler(mediaFiles, logger))
	mux.HandleFunc("PATCH /api/media/{mediaID}", requireAdmin(updateMediaHandler(mediaFiles, logger)))
	mux.HandleFunc("PUT /api/media/{mediaID}/favorite", setFavoriteHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/thumbnail", thumbnailHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/stream", streamHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/playback", playbackHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/hls/{filename}", hlsHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/home", homeHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/progress", getProgressHandler(progress, logger))
	mux.HandleFunc("PUT /api/media/{mediaID}/progress", saveProgressHandler(progress, logger))
	mux.HandleFunc("GET /api/tags", listTagsHandler(tags, logger))
	mux.HandleFunc("POST /api/tags", requireAdmin(createTagHandler(tags, logger)))
	mux.HandleFunc("GET /api/tag-assignments", listTagAssignmentsHandler(tags, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/tags", listMediaTagsHandler(tags, logger))
	mux.HandleFunc("PUT /api/media/{mediaID}/tags", requireAdmin(setMediaTagsHandler(tags, logger)))
	mux.HandleFunc("GET /api/collections", listCollectionsHandler(collections, logger))
	mux.HandleFunc("POST /api/collections", createCollectionHandler(collections, logger))
	mux.HandleFunc("PATCH /api/collections/{collectionID}", updateCollectionHandler(collections, logger))
	mux.HandleFunc("DELETE /api/collections/{collectionID}", deleteCollectionHandler(collections, logger))
	mux.HandleFunc("GET /api/collections/{collectionID}/media", collectionMediaHandler(collections, logger))
	mux.HandleFunc("DELETE /api/collections/{collectionID}/media/{mediaID}", removeCollectionMediaHandler(collections, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/collections", listMediaCollectionsHandler(collections, logger))
	mux.HandleFunc("PUT /api/media/{mediaID}/collections", setMediaCollectionsHandler(collections, logger))
	mux.Handle("/", spaHandler(cfg.WebPath))

	return requestLogger(logger, secureHeaders(requireAuthentication(authentication, logger, mux)))
}

func healthHandler(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(response).Encode(healthResponse{Status: "ok", Service: "flex"})
}

func spaHandler(webPath string) http.Handler {
	fileServer := http.FileServer(http.Dir(webPath))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			http.NotFound(response, request)
			return
		}

		requestedPath := filepath.Join(webPath, filepath.Clean(request.URL.Path))
		if info, err := os.Stat(requestedPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(response, request)
			return
		}

		http.ServeFile(response, request, filepath.Join(webPath, "index.html"))
	})
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		response.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(response, request)
	})
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(response, request)
		logger.Info("request completed",
			"method", request.Method,
			"path", request.URL.Path,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	})
}

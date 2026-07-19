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

	"flex.local/server/internal/config"
	"flex.local/server/internal/library"
	"flex.local/server/internal/media"
	"flex.local/server/internal/playback"
	"flex.local/server/internal/scanmanager"
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
	List(ctx context.Context, libraryID string) ([]media.File, error)
	Get(ctx context.Context, id string) (media.File, error)
	Home(ctx context.Context) (media.Home, error)
	Search(ctx context.Context, query string) ([]media.SearchResult, error)
	Thumbnail(ctx context.Context, id string) (string, error)
	Transcode(ctx context.Context, id string) (string, error)
}

type scanService interface {
	Scan(ctx context.Context, libraryID string) (media.ScanResult, error)
	Trigger(libraryID string)
	Status(libraryID string) scanmanager.Status
}

type playbackService interface {
	Get(ctx context.Context, mediaID string) (playback.Progress, error)
	Save(ctx context.Context, mediaID string, positionMS int64, durationMS int64) (playback.Progress, error)
}

func New(cfg config.Config, logger *slog.Logger, libraries libraryService, mediaFiles mediaService, scans scanService, progress playbackService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler)
	mux.HandleFunc("GET /api/libraries", listLibrariesHandler(libraries, logger))
	mux.HandleFunc("POST /api/libraries", createLibraryHandler(libraries, scans, logger))
	mux.HandleFunc("PATCH /api/libraries/{libraryID}", updateLibraryHandler(libraries, logger))
	mux.HandleFunc("DELETE /api/libraries/{libraryID}", deleteLibraryHandler(libraries, logger))
	mux.HandleFunc("GET /api/libraries/{libraryID}/scan", scanStatusHandler(scans))
	mux.HandleFunc("POST /api/libraries/{libraryID}/scan", scanLibraryHandler(scans, logger))
	mux.HandleFunc("GET /api/media", listMediaHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/search", searchMediaHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}", getMediaHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/thumbnail", thumbnailHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/stream", streamHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/playback", playbackHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/hls/{filename}", hlsHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/home", homeHandler(mediaFiles, logger))
	mux.HandleFunc("GET /api/media/{mediaID}/progress", getProgressHandler(progress, logger))
	mux.HandleFunc("PUT /api/media/{mediaID}/progress", saveProgressHandler(progress, logger))
	mux.Handle("/", spaHandler(cfg.WebPath))

	return requestLogger(logger, secureHeaders(mux))
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

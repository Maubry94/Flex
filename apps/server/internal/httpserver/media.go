package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flex.local/server/internal/library"
	"flex.local/server/internal/media"
)

type mediaResponse struct {
	ID          string     `json:"id"`
	LibraryID   string     `json:"libraryId"`
	Filename    string     `json:"filename"`
	SizeBytes   int64      `json:"sizeBytes"`
	DurationMS  int64      `json:"durationMs"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	Container   string     `json:"container"`
	VideoCodec  string     `json:"videoCodec"`
	AudioCodec  string     `json:"audioCodec"`
	ModifiedAt  time.Time  `json:"modifiedAt"`
	ProgressMS  int64      `json:"progressMs"`
	Completed   bool       `json:"completed"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	RecordedAt  *time.Time `json:"recordedAt"`
	Favorite    bool       `json:"favorite"`
}

type updateMediaRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	RecordedAt  *string `json:"recordedAt"`
	Favorite    bool    `json:"favorite"`
}

type mediaListResponse struct {
	Items []mediaResponse `json:"items"`
}

type mediaSearchResponse struct {
	Items []mediaSearchItemResponse `json:"items"`
}

type mediaSearchItemResponse struct {
	mediaResponse
	LibraryName string `json:"libraryName"`
}

type homeResponse struct {
	ContinueWatching []mediaResponse `json:"continueWatching"`
	RecentlyAdded    []mediaResponse `json:"recentlyAdded"`
}

type scanResponse struct {
	Discovered int `json:"discovered"`
	Indexed    int `json:"indexed"`
	Skipped    int `json:"skipped"`
}

type playbackResponse struct {
	Mode media.PlaybackMode `json:"mode"`
	URL  string             `json:"url"`
}

type folderAssignmentResponse struct {
	MediaID string `json:"mediaId"`
	Folder  string `json:"folder"`
}

type folderAssignmentListResponse struct {
	Items []folderAssignmentResponse `json:"items"`
}

func libraryFoldersHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		items, err := service.Folders(request.Context(), request.PathValue("libraryID"))
		if err != nil {
			logger.Error("list library folders", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de charger les dossiers")
			return
		}
		payload := folderAssignmentListResponse{Items: make([]folderAssignmentResponse, 0, len(items))}
		for _, item := range items {
			payload.Items = append(payload.Items, folderAssignmentResponse{MediaID: item.MediaID, Folder: item.Folder})
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func listMediaHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		libraryID := request.URL.Query().Get("libraryId")
		if libraryID == "" {
			writeError(response, http.StatusBadRequest, "missing_library", "L'identifiant de la bibliothèque est requis")
			return
		}
		items, err := service.List(request.Context(), libraryID)
		if err != nil {
			logger.Error("list media", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de charger les vidéos")
			return
		}
		payload := mediaListResponse{Items: make([]mediaResponse, 0, len(items))}
		for _, item := range items {
			payload.Items = append(payload.Items, mediaToResponse(item))
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func favoritesHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		items, err := service.Favorites(request.Context())
		if err != nil {
			logger.Error("list favorite media", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de charger les favoris")
			return
		}
		payload := mediaListResponse{Items: make([]mediaResponse, 0, len(items))}
		for _, item := range items {
			payload.Items = append(payload.Items, mediaToResponse(item))
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func searchMediaHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		items, err := service.Search(request.Context(), request.URL.Query().Get("q"))
		if err != nil {
			logger.Error("search media", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "La recherche a échoué")
			return
		}
		payload := mediaSearchResponse{Items: make([]mediaSearchItemResponse, 0, len(items))}
		for _, item := range items {
			payload.Items = append(payload.Items, mediaSearchItemResponse{mediaResponse: mediaToResponse(item.File), LibraryName: item.LibraryName})
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func homeHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		home, err := service.Home(request.Context())
		if err != nil {
			logger.Error("load home", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de charger l'accueil")
			return
		}
		payload := homeResponse{
			ContinueWatching: make([]mediaResponse, 0, len(home.ContinueWatching)),
			RecentlyAdded:    make([]mediaResponse, 0, len(home.RecentlyAdded)),
		}
		for _, item := range home.ContinueWatching {
			payload.ContinueWatching = append(payload.ContinueWatching, mediaToResponse(item))
		}
		for _, item := range home.RecentlyAdded {
			payload.RecentlyAdded = append(payload.RecentlyAdded, mediaToResponse(item))
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func getMediaHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		item, err := service.Get(request.Context(), request.PathValue("mediaID"))
		if errors.Is(err, media.ErrNotFound) {
			writeError(response, http.StatusNotFound, "media_not_found", "La vidéo n'existe pas")
			return
		}
		if err != nil {
			logger.Error("get media", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de charger la vidéo")
			return
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(mediaToResponse(item))
	}
}

func updateMediaHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input updateMediaRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeError(response, http.StatusBadRequest, "invalid_request", "La requête est invalide")
			return
		}
		var recordedAt *time.Time
		if input.RecordedAt != nil && *input.RecordedAt != "" {
			parsed, err := time.Parse("2006-01-02", *input.RecordedAt)
			if err != nil {
				writeError(response, http.StatusUnprocessableEntity, "invalid_recorded_at", "La date d'enregistrement est invalide")
				return
			}
			recordedAt = &parsed
		}
		updated, err := service.UpdateMetadata(request.Context(), request.PathValue("mediaID"), media.MetadataInput{
			Title: input.Title, Description: input.Description, RecordedAt: recordedAt, Favorite: input.Favorite,
		})
		switch {
		case errors.Is(err, media.ErrNotFound):
			writeError(response, http.StatusNotFound, "media_not_found", "La vidéo n'existe pas")
		case errors.Is(err, media.ErrInvalidTitle):
			writeError(response, http.StatusUnprocessableEntity, "invalid_title", "Le titre est requis et doit contenir moins de 200 caractères")
		case err != nil:
			logger.Error("update media metadata", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de modifier la vidéo")
		default:
			response.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(response).Encode(mediaToResponse(updated))
		}
	}
}

func thumbnailHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		path, err := service.Thumbnail(request.Context(), request.PathValue("mediaID"))
		if errors.Is(err, media.ErrNotFound) {
			http.NotFound(response, request)
			return
		}
		if err != nil {
			logger.Error("generate thumbnail", "error", err)
			http.Error(response, "thumbnail unavailable", http.StatusInternalServerError)
			return
		}
		response.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFile(response, request, path)
	}
}

func streamHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		item, err := service.Get(request.Context(), request.PathValue("mediaID"))
		if errors.Is(err, media.ErrNotFound) {
			http.NotFound(response, request)
			return
		}
		if err != nil {
			logger.Error("get media stream", "error", err)
			http.Error(response, "stream unavailable", http.StatusInternalServerError)
			return
		}
		file, err := os.Open(item.Path)
		if err != nil {
			logger.Error("open media stream", "error", err)
			http.Error(response, "stream unavailable", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			http.Error(response, "stream unavailable", http.StatusInternalServerError)
			return
		}
		http.ServeContent(response, request, item.Filename, info.ModTime(), file)
	}
}

func playbackHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		mediaID := request.PathValue("mediaID")
		item, err := service.Get(request.Context(), mediaID)
		if errors.Is(err, media.ErrNotFound) {
			writeError(response, http.StatusNotFound, "media_not_found", "La vidéo n'existe pas")
			return
		}
		if err != nil {
			logger.Error("get playback mode", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de préparer la lecture")
			return
		}
		mode := media.SelectPlaybackMode(item)
		url := "/api/media/" + mediaID + "/stream"
		if mode == media.PlaybackHLS {
			url = "/api/media/" + mediaID + "/hls/index.m3u8"
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(playbackResponse{Mode: mode, URL: url})
	}
}

func hlsHandler(service mediaService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		filename := request.PathValue("filename")
		if !validHLSFilename(filename) {
			http.NotFound(response, request)
			return
		}
		directory, err := service.Transcode(request.Context(), request.PathValue("mediaID"))
		if errors.Is(err, media.ErrNotFound) {
			http.NotFound(response, request)
			return
		}
		if err != nil {
			logger.Error("transcode media", "error", err)
			http.Error(response, "transcode unavailable", http.StatusInternalServerError)
			return
		}
		if filename == "index.m3u8" {
			response.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			response.Header().Set("Cache-Control", "no-cache")
		} else {
			response.Header().Set("Content-Type", "video/mp2t")
			response.Header().Set("Cache-Control", "public, max-age=86400")
		}
		http.ServeFile(response, request, filepath.Join(directory, filename))
	}
}

func validHLSFilename(filename string) bool {
	if filename == "index.m3u8" {
		return true
	}
	if !strings.HasPrefix(filename, "segment-") || !strings.HasSuffix(filename, ".ts") {
		return false
	}
	digits := strings.TrimSuffix(strings.TrimPrefix(filename, "segment-"), ".ts")
	if digits == "" {
		return false
	}
	for _, character := range digits {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func scanLibraryHandler(service scanService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		result, err := service.Scan(request.Context(), request.PathValue("libraryID"))
		if errors.Is(err, library.ErrNotFound) {
			writeError(response, http.StatusNotFound, "library_not_found", "La bibliothèque n'existe pas")
			return
		}
		if err != nil {
			logger.Error("scan library", "error", err)
			writeError(response, http.StatusInternalServerError, "scan_failed", "L'analyse de la bibliothèque a échoué")
			return
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(scanResponse{
			Discovered: result.Discovered,
			Indexed:    result.Indexed,
			Skipped:    result.Skipped,
		})
	}
}

func scanStatusHandler(service scanService) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		status := service.Status(request.PathValue("libraryID"))
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(struct {
			State     string     `json:"state"`
			StartedAt *time.Time `json:"startedAt,omitempty"`
			LastError string     `json:"lastError,omitempty"`
		}{State: status.State, StartedAt: status.StartedAt, LastError: status.LastError})
	}
}

func mediaToResponse(item media.File) mediaResponse {
	return mediaResponse{
		ID: item.ID, LibraryID: item.LibraryID, Filename: item.Filename, SizeBytes: item.SizeBytes,
		DurationMS: item.DurationMS, Width: item.Width, Height: item.Height, Container: item.Container,
		VideoCodec: item.VideoCodec, AudioCodec: item.AudioCodec, ModifiedAt: item.ModifiedAt,
		ProgressMS: item.ProgressMS, Completed: item.Completed,
		Title: item.Title, Description: item.Description, RecordedAt: item.RecordedAt, Favorite: item.Favorite,
	}
}

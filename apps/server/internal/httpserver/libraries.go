package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"flex.local/server/internal/library"
)

type libraryResponse struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Path               string     `json:"path"`
	CreatedAt          time.Time  `json:"createdAt"`
	LastScanAt         *time.Time `json:"lastScanAt"`
	LastScanDiscovered int        `json:"lastScanDiscovered"`
	LastScanIndexed    int        `json:"lastScanIndexed"`
	LastScanUnchanged  int        `json:"lastScanUnchanged"`
	LastScanSkipped    int        `json:"lastScanSkipped"`
}

type listLibrariesResponse struct {
	Items []libraryResponse `json:"items"`
}

type createLibraryRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func listLibrariesHandler(service libraryService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		items, err := service.List(request.Context())
		if err != nil {
			logger.Error("list libraries", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de charger les bibliothèques")
			return
		}

		payload := listLibrariesResponse{Items: make([]libraryResponse, 0, len(items))}
		for _, item := range items {
			payload.Items = append(payload.Items, libraryToResponse(item))
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func createLibraryHandler(service libraryService, scans scanService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input createLibraryRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeError(response, http.StatusBadRequest, "invalid_request", "La requête est invalide")
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeError(response, http.StatusBadRequest, "invalid_request", "La requête doit contenir un seul objet JSON")
			return
		}

		created, err := service.Add(request.Context(), input.Name, input.Path)
		switch {
		case errors.Is(err, library.ErrInvalidName):
			writeError(response, http.StatusUnprocessableEntity, "invalid_name", "Le nom de la bibliothèque est requis")
			return
		case errors.Is(err, library.ErrInvalidPath):
			writeError(response, http.StatusUnprocessableEntity, "invalid_path", "Le dossier doit exister dans le répertoire des médias")
			return
		case errors.Is(err, library.ErrConflict):
			writeError(response, http.StatusConflict, "path_conflict", "Ce dossier appartient déjà à une bibliothèque")
			return
		case err != nil:
			logger.Error("create library", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de créer la bibliothèque")
			return
		}
		scans.Trigger(created.ID)

		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		response.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(response).Encode(libraryToResponse(created))
	}
}

func updateLibraryHandler(service libraryService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input createLibraryRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeError(response, http.StatusBadRequest, "invalid_request", "La requête est invalide")
			return
		}
		updated, err := service.Update(request.Context(), request.PathValue("libraryID"), input.Name, input.Path)
		switch {
		case errors.Is(err, library.ErrNotFound):
			writeError(response, http.StatusNotFound, "library_not_found", "La bibliothèque n'existe pas")
		case errors.Is(err, library.ErrInvalidName):
			writeError(response, http.StatusUnprocessableEntity, "invalid_name", "Le nom de la bibliothèque est requis")
		case errors.Is(err, library.ErrInvalidPath):
			writeError(response, http.StatusUnprocessableEntity, "invalid_path", "Le dossier doit exister dans le répertoire des médias")
		case errors.Is(err, library.ErrConflict):
			writeError(response, http.StatusConflict, "path_conflict", "Ce dossier appartient déjà à une bibliothèque")
		case err != nil:
			logger.Error("update library", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de modifier la bibliothèque")
		default:
			response.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(response).Encode(libraryToResponse(updated))
		}
	}
}

func deleteLibraryHandler(service libraryService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		err := service.Delete(request.Context(), request.PathValue("libraryID"))
		if errors.Is(err, library.ErrNotFound) {
			writeError(response, http.StatusNotFound, "library_not_found", "La bibliothèque n'existe pas")
			return
		}
		if err != nil {
			logger.Error("delete library", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de supprimer la bibliothèque")
			return
		}
		response.WriteHeader(http.StatusNoContent)
	}
}

func libraryToResponse(item library.Library) libraryResponse {
	return libraryResponse{
		ID: item.ID, Name: item.Name, Path: item.Path, CreatedAt: item.CreatedAt,
		LastScanAt: item.LastScanAt, LastScanDiscovered: item.LastScanDiscovered,
		LastScanIndexed: item.LastScanIndexed, LastScanUnchanged: item.LastScanUnchanged, LastScanSkipped: item.LastScanSkipped,
	}
}

func writeError(response http.ResponseWriter, status int, code string, message string) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(errorResponse{Code: code, Message: message})
}

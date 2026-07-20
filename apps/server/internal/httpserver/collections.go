package httpserver

import (
	"encoding/json"
	"errors"
	"flex.local/server/internal/collection"
	"log/slog"
	"net/http"
)

type collectionResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	MediaCount int    `json:"mediaCount"`
}
type collectionListResponse struct {
	Items []collectionResponse `json:"items"`
}
type collectionRequest struct {
	Name string `json:"name"`
}
type setCollectionsRequest struct {
	CollectionIDs []string `json:"collectionIds"`
}
type mediaIDsResponse struct {
	MediaIDs []string `json:"mediaIds"`
}

func listCollectionsHandler(service collectionService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := service.List(r.Context(), currentUser(r).ID)
		writeCollections(w, items, err, logger)
	}
}
func createCollectionHandler(service collectionService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input collectionRequest
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
		d.DisallowUnknownFields()
		if d.Decode(&input) != nil {
			writeError(w, 400, "invalid_request", "La requête est invalide")
			return
		}
		item, err := service.Create(r.Context(), currentUser(r).ID, input.Name)
		if errors.Is(err, collection.ErrInvalid) {
			writeError(w, 422, "invalid_collection", "Le nom est invalide")
			return
		}
		if errors.Is(err, collection.ErrConflict) {
			writeError(w, 409, "collection_conflict", "Cette collection existe déjà")
			return
		}
		if err != nil {
			logger.Error("create collection", "error", err)
			writeError(w, 500, "internal_error", "Impossible de créer la collection")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(toCollection(item))
	}
}
func updateCollectionHandler(service collectionService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input collectionRequest
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
		d.DisallowUnknownFields()
		if d.Decode(&input) != nil {
			writeError(w, 400, "invalid_request", "La requête est invalide")
			return
		}
		item, err := service.Update(r.Context(), currentUser(r).ID, r.PathValue("collectionID"), input.Name)
		if writeCollectionError(w, err, logger) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toCollection(item))
	}
}
func deleteCollectionHandler(service collectionService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeCollectionError(w, service.Delete(r.Context(), currentUser(r).ID, r.PathValue("collectionID")), logger) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
func removeCollectionMediaHandler(service collectionService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeCollectionError(w, service.RemoveMedia(r.Context(), currentUser(r).ID, r.PathValue("collectionID"), r.PathValue("mediaID")), logger) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
func listMediaCollectionsHandler(service collectionService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := service.ListForMedia(r.Context(), currentUser(r).ID, r.PathValue("mediaID"))
		writeCollections(w, items, err, logger)
	}
}
func setMediaCollectionsHandler(service collectionService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input setCollectionsRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&input) != nil || input.CollectionIDs == nil {
			writeError(w, 400, "invalid_request", "La requête est invalide")
			return
		}
		items, err := service.SetForMedia(r.Context(), currentUser(r).ID, r.PathValue("mediaID"), input.CollectionIDs)
		writeCollections(w, items, err, logger)
	}
}
func collectionMediaHandler(service collectionService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, err := service.MediaIDs(r.Context(), currentUser(r).ID, r.PathValue("collectionID"))
		if err != nil {
			logger.Error("list collection media", "error", err)
			writeError(w, 500, "internal_error", "Impossible de charger la collection")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mediaIDsResponse{MediaIDs: ids})
	}
}
func writeCollections(w http.ResponseWriter, items []collection.Collection, err error, logger *slog.Logger) {
	if errors.Is(err, collection.ErrMediaNotFound) {
		writeError(w, 404, "media_not_found", "La vidéo n'existe pas")
		return
	}
	if errors.Is(err, collection.ErrInvalid) {
		writeError(w, 422, "invalid_collections", "La sélection est invalide")
		return
	}
	if err != nil {
		logger.Error("collections", "error", err)
		writeError(w, 500, "internal_error", "Impossible de charger les collections")
		return
	}
	payload := collectionListResponse{Items: make([]collectionResponse, 0, len(items))}
	for _, item := range items {
		payload.Items = append(payload.Items, toCollection(item))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
func toCollection(item collection.Collection) collectionResponse {
	return collectionResponse{ID: item.ID, Name: item.Name, MediaCount: item.MediaCount}
}

func writeCollectionError(w http.ResponseWriter, err error, logger *slog.Logger) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, collection.ErrInvalid) {
		writeError(w, 422, "invalid_collection", "Le nom est invalide")
		return true
	}
	if errors.Is(err, collection.ErrConflict) {
		writeError(w, 409, "collection_conflict", "Cette collection existe déjà")
		return true
	}
	if errors.Is(err, collection.ErrNotFound) {
		writeError(w, 404, "collection_not_found", "La collection ou la vidéo n'existe pas")
		return true
	}
	logger.Error("manage collection", "error", err)
	writeError(w, 500, "internal_error", "Impossible de modifier la collection")
	return true
}

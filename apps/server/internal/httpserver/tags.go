package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"flex.local/server/internal/tag"
)

type tagResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type tagListResponse struct {
	Items []tagResponse `json:"items"`
}

type tagAssignmentResponse struct {
	MediaID string      `json:"mediaId"`
	Tag     tagResponse `json:"tag"`
}

type tagAssignmentListResponse struct {
	Items []tagAssignmentResponse `json:"items"`
}

type createTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type setMediaTagsRequest struct {
	TagIDs []string `json:"tagIds"`
}

func listTagsHandler(service tagService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		items, err := service.List(request.Context())
		writeTags(response, items, err, logger, "list tags")
	}
}

func listTagAssignmentsHandler(service tagService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		items, err := service.Assignments(request.Context())
		if err != nil {
			logger.Error("list tag assignments", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de charger les attributions de tags")
			return
		}
		payload := tagAssignmentListResponse{Items: make([]tagAssignmentResponse, 0, len(items))}
		for _, item := range items {
			payload.Items = append(payload.Items, tagAssignmentResponse{MediaID: item.MediaID, Tag: toTagResponse(item.Tag)})
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func createTagHandler(service tagService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input createTagRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<16))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeError(response, http.StatusBadRequest, "invalid_request", "La requête est invalide")
			return
		}
		item, err := service.Create(request.Context(), input.Name, input.Color)
		switch {
		case errors.Is(err, tag.ErrInvalid):
			writeError(response, http.StatusUnprocessableEntity, "invalid_tag", "Le nom ou la couleur du tag est invalide")
		case errors.Is(err, tag.ErrConflict):
			writeError(response, http.StatusConflict, "tag_conflict", "Un tag avec ce nom existe déjà")
		case err != nil:
			logger.Error("create tag", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de créer le tag")
		default:
			response.Header().Set("Content-Type", "application/json; charset=utf-8")
			response.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(response).Encode(toTagResponse(item))
		}
	}
}

func listMediaTagsHandler(service tagService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		items, err := service.ListForMedia(request.Context(), request.PathValue("mediaID"))
		writeTags(response, items, err, logger, "list media tags")
	}
}

func setMediaTagsHandler(service tagService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input setMediaTagsRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<16))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil || input.TagIDs == nil {
			writeError(response, http.StatusBadRequest, "invalid_request", "La requête est invalide")
			return
		}
		items, err := service.SetForMedia(request.Context(), request.PathValue("mediaID"), input.TagIDs)
		writeTags(response, items, err, logger, "set media tags")
	}
}

func writeTags(response http.ResponseWriter, items []tag.Tag, err error, logger *slog.Logger, operation string) {
	switch {
	case errors.Is(err, tag.ErrMediaNotFound):
		writeError(response, http.StatusNotFound, "media_not_found", "La vidéo n'existe pas")
	case errors.Is(err, tag.ErrInvalid):
		writeError(response, http.StatusUnprocessableEntity, "invalid_tags", "La sélection de tags est invalide")
	case err != nil:
		logger.Error(operation, "error", err)
		writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de charger les tags")
	default:
		payload := tagListResponse{Items: make([]tagResponse, 0, len(items))}
		for _, item := range items {
			payload.Items = append(payload.Items, toTagResponse(item))
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func toTagResponse(item tag.Tag) tagResponse {
	return tagResponse{ID: item.ID, Name: item.Name, Color: item.Color}
}

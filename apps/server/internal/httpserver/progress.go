package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"flex.local/server/internal/playback"
)

type progressResponse struct {
	MediaID    string    `json:"mediaId"`
	PositionMS int64     `json:"positionMs"`
	DurationMS int64     `json:"durationMs"`
	Completed  bool      `json:"completed"`
	UpdatedAt  time.Time `json:"updatedAt,omitempty"`
}

type saveProgressRequest struct {
	PositionMS int64 `json:"positionMs"`
	DurationMS int64 `json:"durationMs"`
}

func getProgressHandler(service playbackService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		progress, err := service.Get(request.Context(), request.PathValue("mediaID"))
		if err != nil {
			logger.Error("get playback progress", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de charger la progression")
			return
		}
		writeProgress(response, progress)
	}
}

func saveProgressHandler(service playbackService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input saveProgressRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<16))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeError(response, http.StatusBadRequest, "invalid_request", "La progression est invalide")
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeError(response, http.StatusBadRequest, "invalid_request", "La requête doit contenir un seul objet JSON")
			return
		}
		progress, err := service.Save(request.Context(), request.PathValue("mediaID"), input.PositionMS, input.DurationMS)
		if err != nil {
			logger.Error("save playback progress", "error", err)
			writeError(response, http.StatusInternalServerError, "internal_error", "Impossible de sauvegarder la progression")
			return
		}
		writeProgress(response, progress)
	}
}

func writeProgress(response http.ResponseWriter, progress playback.Progress) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(response).Encode(progressResponse{
		MediaID: progress.MediaID, PositionMS: progress.PositionMS, DurationMS: progress.DurationMS,
		Completed: progress.Completed, UpdatedAt: progress.UpdatedAt,
	})
}

package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"flex.local/server/internal/auth"
)

type userListResponse struct {
	Items []authUserResponse `json:"items"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type updateUserRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func listUsersHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		users, err := service.ListUsers(request.Context())
		if err != nil {
			logger.Error("list users", "error", err)
			writeError(response, 500, "internal_error", "Impossible de charger les utilisateurs")
			return
		}
		payload := userListResponse{Items: make([]authUserResponse, 0, len(users))}
		for _, user := range users {
			payload.Items = append(payload.Items, userResponse(user))
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func createUserHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input createUserRequest
		if !decodeAuthJSON(response, request, &input) {
			return
		}
		user, err := service.CreateUser(request.Context(), input.Username, input.Password, input.Role)
		if writeUserError(response, err, logger) {
			return
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		response.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(response).Encode(userResponse(user))
	}
}

func updateUserHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input updateUserRequest
		if !decodeAuthJSON(response, request, &input) {
			return
		}
		user, err := service.UpdateUser(request.Context(), request.PathValue("userID"), auth.UserInput{Username: input.Username, Role: input.Role, Active: input.Active})
		if writeUserError(response, err, logger) {
			return
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(userResponse(user))
	}
}

func resetUserPasswordHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input resetPasswordRequest
		if !decodeAuthJSON(response, request, &input) {
			return
		}
		if writeUserError(response, service.ResetPassword(request.Context(), request.PathValue("userID"), input.Password), logger) {
			return
		}
		response.WriteHeader(http.StatusNoContent)
	}
}

func deleteUserHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if writeUserError(response, service.DeleteUser(request.Context(), request.PathValue("userID")), logger) {
			return
		}
		response.WriteHeader(http.StatusNoContent)
	}
}

func decodeAuthJSON(response http.ResponseWriter, request *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<16))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		writeError(response, 400, "invalid_request", "La requête est invalide")
		return false
	}
	return true
}

func writeUserError(response http.ResponseWriter, err error, logger *slog.Logger) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		writeError(response, 422, "invalid_user", "Les informations de l’utilisateur sont invalides")
	case errors.Is(err, auth.ErrConflict):
		writeError(response, 409, "username_conflict", "Ce nom d’utilisateur existe déjà")
	case errors.Is(err, auth.ErrNotFound):
		writeError(response, 404, "user_not_found", "L’utilisateur n’existe pas")
	case errors.Is(err, auth.ErrLastAdmin):
		writeError(response, 409, "last_admin", "Le dernier administrateur actif doit être conservé")
	default:
		logger.Error("manage user", "error", err)
		writeError(response, 500, "internal_error", "Impossible de modifier l’utilisateur")
	}
	return true
}

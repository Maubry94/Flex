package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"flex.local/server/internal/auth"
)

const sessionCookieName = "flex_session"

type authService interface {
	Configured(context.Context) (bool, error)
	Setup(context.Context, string, string) (auth.Session, error)
	Login(context.Context, string, string) (auth.Session, error)
	Authenticate(context.Context, string) (auth.User, error)
	Logout(context.Context, string) error
	ChangePassword(context.Context, string, string, string) (auth.Session, error)
	UpdateProfile(context.Context, string, string) (auth.User, error)
	ListUsers(context.Context) ([]auth.User, error)
	CreateUser(context.Context, string, string, string) (auth.User, error)
	UpdateUser(context.Context, string, auth.UserInput) (auth.User, error)
	ResetPassword(context.Context, string, string) error
	DeleteUser(context.Context, string) error
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type updateProfileRequest struct {
	Username string `json:"username"`
}

type authUserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

type authStatusResponse struct {
	Configured    bool              `json:"configured"`
	Authenticated bool              `json:"authenticated"`
	User          *authUserResponse `json:"user,omitempty"`
}

type authContextKey struct{}

func authStatusHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		configured, err := service.Configured(request.Context())
		if err != nil {
			logger.Error("authentication status", "error", err)
			writeError(response, 500, "internal_error", "Impossible de vérifier la configuration")
			return
		}
		payload := authStatusResponse{Configured: configured}
		if configured {
			if user, err := authenticateRequest(request, service); err == nil {
				formatted := userResponse(user)
				payload.Authenticated = true
				payload.User = &formatted
			}
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func setupHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input authRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<16))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil {
			writeError(response, 400, "invalid_request", "La requête est invalide")
			return
		}
		session, err := service.Setup(request.Context(), input.Username, input.Password)
		switch {
		case errors.Is(err, auth.ErrAlreadyConfigured):
			writeError(response, 409, "already_configured", "Flex est déjà configuré")
			return
		case errors.Is(err, auth.ErrInvalidInput):
			writeError(response, 422, "invalid_credentials", "Vérifiez le nom d’utilisateur et le mot de passe")
			return
		case err != nil:
			logger.Error("create administrator", "error", err)
			writeError(response, 500, "internal_error", "Impossible de créer l’administrateur")
			return
		}
		setSessionCookie(response, request, session)
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		response.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(response).Encode(userResponse(session.User))
	}
}

func loginHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input authRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<16))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil {
			writeError(response, 400, "invalid_request", "La requête est invalide")
			return
		}
		session, err := service.Login(request.Context(), input.Username, input.Password)
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(response, 401, "invalid_credentials", "Nom d’utilisateur ou mot de passe incorrect")
			return
		}
		if err != nil {
			logger.Error("login", "error", err)
			writeError(response, 500, "internal_error", "Impossible de se connecter")
			return
		}
		setSessionCookie(response, request, session)
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(response).Encode(userResponse(session.User))
	}
}

func logoutHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		token := sessionToken(request)
		if err := service.Logout(request.Context(), token); err != nil {
			logger.Error("logout", "error", err)
			writeError(response, 500, "internal_error", "Impossible de se déconnecter")
			return
		}
		http.SetCookie(response, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: requestIsHTTPS(request), SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(0, 0)})
		response.WriteHeader(http.StatusNoContent)
	}
}

func changePasswordHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input changePasswordRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<16))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil {
			writeError(response, 400, "invalid_request", "La requête est invalide")
			return
		}
		session, err := service.ChangePassword(request.Context(), currentUser(request).ID, input.CurrentPassword, input.NewPassword)
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			writeError(response, 401, "invalid_credentials", "Le mot de passe actuel est incorrect")
		case errors.Is(err, auth.ErrInvalidInput):
			writeError(response, 422, "invalid_password", "Le nouveau mot de passe doit contenir au moins 12 caractères")
		case err != nil:
			logger.Error("change password", "error", err)
			writeError(response, 500, "internal_error", "Impossible de modifier le mot de passe")
		default:
			setSessionCookie(response, request, session)
			response.WriteHeader(http.StatusNoContent)
		}
	}
}

func updateProfileHandler(service authService, logger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var input updateProfileRequest
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<16))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil {
			writeError(response, 400, "invalid_request", "La requête est invalide")
			return
		}
		user, err := service.UpdateProfile(request.Context(), currentUser(request).ID, input.Username)
		switch {
		case errors.Is(err, auth.ErrInvalidInput):
			writeError(response, 422, "invalid_username", "Le nom d’utilisateur est invalide")
		case errors.Is(err, auth.ErrConflict):
			writeError(response, 409, "username_conflict", "Ce nom d’utilisateur existe déjà")
		case err != nil:
			logger.Error("update profile", "error", err)
			writeError(response, 500, "internal_error", "Impossible de modifier le profil")
		default:
			response.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(response).Encode(userResponse(user))
		}
	}
}

func requireAuthentication(service authService, logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.URL.Path, "/api/") || publicAPI(request) {
			next.ServeHTTP(response, request)
			return
		}
		configured, err := service.Configured(request.Context())
		if err != nil {
			logger.Error("authorize request", "error", err)
			writeError(response, 500, "internal_error", "Impossible de vérifier la session")
			return
		}
		if !configured {
			writeError(response, http.StatusPreconditionRequired, "setup_required", "La configuration initiale est requise")
			return
		}
		user, err := authenticateRequest(request, service)
		if err != nil {
			writeError(response, http.StatusUnauthorized, "unauthenticated", "Une connexion est requise")
			return
		}
		next.ServeHTTP(response, request.WithContext(context.WithValue(request.Context(), authContextKey{}, user)))
	})
}

func publicAPI(request *http.Request) bool {
	if request.Method == http.MethodGet && (request.URL.Path == "/api/health" || request.URL.Path == "/api/auth/status") {
		return true
	}
	return request.Method == http.MethodPost && (request.URL.Path == "/api/auth/setup" || request.URL.Path == "/api/auth/login" || request.URL.Path == "/api/auth/logout")
}

func authenticateRequest(request *http.Request, service authService) (auth.User, error) {
	return service.Authenticate(request.Context(), sessionToken(request))
}

func sessionToken(request *http.Request) string {
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setSessionCookie(response http.ResponseWriter, request *http.Request, session auth.Session) {
	http.SetCookie(response, &http.Cookie{Name: sessionCookieName, Value: session.Token, Path: "/", HttpOnly: true, Secure: requestIsHTTPS(request), SameSite: http.SameSiteLaxMode, Expires: session.ExpiresAt, MaxAge: int(time.Until(session.ExpiresAt).Seconds())})
}

func requestIsHTTPS(request *http.Request) bool {
	return request.TLS != nil || strings.EqualFold(strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Proto"), ",")[0]), "https")
}

func userResponse(user auth.User) authUserResponse {
	return authUserResponse{ID: user.ID, Username: user.Username, Role: user.Role, Active: user.Active}
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		user, ok := request.Context().Value(authContextKey{}).(auth.User)
		if !ok || user.Role != "admin" {
			writeError(response, http.StatusForbidden, "forbidden", "Cette action est réservée aux administrateurs")
			return
		}
		next(response, request)
	}
}

func currentUser(request *http.Request) auth.User {
	user, _ := request.Context().Value(authContextKey{}).(auth.User)
	return user
}

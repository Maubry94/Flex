package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	passwordIterations = 600_000
	sessionDuration    = 30 * 24 * time.Hour
)

var (
	ErrAlreadyConfigured  = errors.New("authentication already configured")
	ErrInvalidInput       = errors.New("invalid authentication input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("unauthenticated")
)

type User struct {
	ID       string
	Username string
	Role     string
	Active   bool
}

type Session struct {
	Token     string
	ExpiresAt time.Time
	User      User
}

type Service struct {
	db         *sql.DB
	iterations int
	now        func() time.Time
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, iterations: passwordIterations, now: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) Configured(ctx context.Context) (bool, error) {
	var configured bool
	err := service.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin' AND active = 1)`).Scan(&configured)
	return configured, err
}

func (service *Service) Setup(ctx context.Context, username string, password string) (Session, error) {
	username, err := validateCredentials(username, password)
	if err != nil {
		return Session{}, err
	}
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback()
	var configured bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users)`).Scan(&configured); err != nil {
		return Session{}, err
	}
	if configured {
		return Session{}, ErrAlreadyConfigured
	}
	passwordHash, err := hashPassword(password, service.iterations)
	if err != nil {
		return Session{}, err
	}
	id, err := randomHex(16)
	if err != nil {
		return Session{}, err
	}
	now := service.now()
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, username, display_name, password_hash, role, active, created_at, updated_at) VALUES (?, ?, ?, ?, 'admin', 1, ?, ?)`, id, username, username, passwordHash, formatTime(now), formatTime(now)); err != nil {
		return Session{}, fmt.Errorf("create administrator: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO playback_progress (profile_id, media_id, position_ms, duration_ms, completed, updated_at) SELECT ?, media_id, position_ms, duration_ms, completed, updated_at FROM playback_progress WHERE profile_id = 'local'`, id); err != nil {
		return Session{}, fmt.Errorf("claim legacy playback progress: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO user_media_state (user_id, media_id, favorite, updated_at) SELECT ?, media_id, favorite, updated_at FROM media_metadata WHERE favorite = 1`, id); err != nil {
		return Session{}, fmt.Errorf("claim legacy favorites: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE collections SET user_id = ? WHERE user_id = 'local'`, id); err != nil {
		return Session{}, fmt.Errorf("claim legacy collections: %w", err)
	}
	session, err := createSession(ctx, tx, User{ID: id, Username: username, Role: "admin", Active: true}, now)
	if err != nil {
		return Session{}, err
	}
	if err := tx.Commit(); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (service *Service) Login(ctx context.Context, username string, password string) (Session, error) {
	username = strings.TrimSpace(username)
	var user User
	var passwordHash string
	var active int
	err := service.db.QueryRowContext(ctx, `SELECT id, username, password_hash, role, active FROM users WHERE username = ?`, username).Scan(&user.ID, &user.Username, &passwordHash, &user.Role, &active)
	if errors.Is(err, sql.ErrNoRows) {
		_ = pbkdf2SHA256([]byte(password), make([]byte, 16), service.iterations, 32)
		return Session{}, ErrInvalidCredentials
	}
	if err == nil && (active != 1 || !verifyPassword(password, passwordHash)) {
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}
	user.Active = true
	return createSession(ctx, service.db, user, service.now())
}

func (service *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrUnauthenticated
	}
	hash := tokenHash(token)
	var user User
	var active int
	var expiresAt string
	err := service.db.QueryRowContext(ctx, `SELECT u.id, u.username, u.role, u.active, s.expires_at FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = ?`, hash).Scan(&user.ID, &user.Username, &user.Role, &active, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUnauthenticated
	}
	if err != nil {
		return User{}, err
	}
	expires, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil || active != 1 || !expires.After(service.now()) {
		_, _ = service.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hash)
		return User{}, ErrUnauthenticated
	}
	user.Active = true
	return user, nil
}

func (service *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := service.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash(token))
	return err
}

func (service *Service) ChangePassword(ctx context.Context, userID string, currentPassword string, newPassword string) (Session, error) {
	if !validPassword(newPassword) {
		return Session{}, ErrInvalidInput
	}
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback()
	var user User
	var passwordHash string
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT id, username, password_hash, role, active FROM users WHERE id = ?`, userID).Scan(&user.ID, &user.Username, &passwordHash, &user.Role, &active); err != nil {
		return Session{}, err
	}
	if active != 1 || !verifyPassword(currentPassword, passwordHash) {
		return Session{}, ErrInvalidCredentials
	}
	newHash, err := hashPassword(newPassword, service.iterations)
	if err != nil {
		return Session{}, err
	}
	now := service.now()
	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, newHash, formatTime(now), userID); err != nil {
		return Session{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return Session{}, err
	}
	user.Active = true
	session, err := createSession(ctx, tx, user, now)
	if err != nil {
		return Session{}, err
	}
	if err := tx.Commit(); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (service *Service) UpdateProfile(ctx context.Context, userID string, username string) (User, error) {
	username, err := validateUsername(username)
	if err != nil {
		return User{}, ErrInvalidInput
	}
	result, err := service.db.ExecContext(ctx, `UPDATE users SET username = ?, display_name = ?, updated_at = ? WHERE id = ?`, username, username, formatTime(service.now()), userID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	if count, err := result.RowsAffected(); err != nil {
		return User{}, err
	} else if count == 0 {
		return User{}, ErrNotFound
	}
	var user User
	var active int
	if err := service.db.QueryRowContext(ctx, `SELECT id, username, role, active FROM users WHERE id = ?`, userID).Scan(&user.ID, &user.Username, &user.Role, &active); err != nil {
		return User{}, err
	}
	user.Active = active == 1
	return user, nil
}

type queryExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func createSession(ctx context.Context, executor queryExecutor, user User, now time.Time) (Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	expiresAt := now.Add(sessionDuration)
	if _, err := executor.ExecContext(ctx, `INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`, tokenHash(token), user.ID, formatTime(now), formatTime(expiresAt)); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return Session{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

func validateCredentials(username string, password string) (string, error) {
	username, err := validateUsername(username)
	if err != nil || !validPassword(password) {
		return "", ErrInvalidInput
	}
	return username, nil
}

func hashPassword(password string, iterations int) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derived := pbkdf2SHA256([]byte(password), salt, iterations, 32)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iterations, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(derived)), nil
}

func verifyPassword(password string, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 100_000 || iterations > 2_000_000 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(expected) != 32 {
		return false
	}
	actual := pbkdf2SHA256([]byte(password), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func pbkdf2SHA256(password []byte, salt []byte, iterations int, length int) []byte {
	result := make([]byte, 0, length)
	for block := uint32(1); len(result) < length; block++ {
		message := append(append([]byte{}, salt...), byte(block>>24), byte(block>>16), byte(block>>8), byte(block))
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(message)
		u := mac.Sum(nil)
		t := append([]byte{}, u...)
		for iteration := 1; iteration < iterations; iteration++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for index := range t {
				t[index] ^= u[index]
			}
		}
		result = append(result, t...)
	}
	return result[:length]
}

func tokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func randomHex(length int) (string, error) {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

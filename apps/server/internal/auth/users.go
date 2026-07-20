package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrConflict  = errors.New("username already exists")
	ErrNotFound  = errors.New("user not found")
	ErrLastAdmin = errors.New("last active administrator")
)

type UserInput struct {
	Username string
	Role     string
	Active   bool
}

func (service *Service) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := service.db.QueryContext(ctx, `SELECT id, username, role, active FROM users ORDER BY username COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	users := make([]User, 0)
	for rows.Next() {
		var user User
		var active int
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &active); err != nil {
			return nil, err
		}
		user.Active = active == 1
		users = append(users, user)
	}
	return users, rows.Err()
}

func (service *Service) CreateUser(ctx context.Context, username string, password string, role string) (User, error) {
	username, err := validateUsername(username)
	if err != nil || !validPassword(password) || !validRole(role) {
		return User{}, ErrInvalidInput
	}
	passwordHash, err := hashPassword(password, service.iterations)
	if err != nil {
		return User{}, err
	}
	id, err := randomHex(16)
	if err != nil {
		return User{}, err
	}
	now := formatTime(service.now())
	_, err = service.db.ExecContext(ctx, `INSERT INTO users (id, username, display_name, password_hash, role, active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 1, ?, ?)`, id, username, username, passwordHash, role, now, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return User{}, ErrConflict
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return User{ID: id, Username: username, Role: role, Active: true}, nil
}

func (service *Service) UpdateUser(ctx context.Context, id string, input UserInput) (User, error) {
	username, err := validateUsername(input.Username)
	if err != nil || !validRole(input.Role) {
		return User{}, ErrInvalidInput
	}
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()
	current, err := getUser(ctx, tx, id)
	if err != nil {
		return User{}, err
	}
	if current.Role == "admin" && current.Active && (input.Role != "admin" || !input.Active) {
		if err := ensureAnotherAdmin(ctx, tx, id); err != nil {
			return User{}, err
		}
	}
	active := 0
	if input.Active {
		active = 1
	}
	_, err = tx.ExecContext(ctx, `UPDATE users SET username = ?, display_name = ?, role = ?, active = ?, updated_at = ? WHERE id = ?`, username, username, input.Role, active, formatTime(service.now()), id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return User{}, ErrConflict
		}
		return User{}, fmt.Errorf("update user: %w", err)
	}
	if !input.Active {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, id); err != nil {
			return User{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return User{}, err
	}
	return User{ID: id, Username: username, Role: input.Role, Active: input.Active}, nil
}

func (service *Service) ResetPassword(ctx context.Context, id string, password string) error {
	if !validPassword(password) {
		return ErrInvalidInput
	}
	passwordHash, err := hashPassword(password, service.iterations)
	if err != nil {
		return err
	}
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, passwordHash, formatTime(service.now()), id)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil {
		return err
	} else if count == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (service *Service) DeleteUser(ctx context.Context, id string) error {
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user, err := getUser(ctx, tx, id)
	if err != nil {
		return err
	}
	if user.Role == "admin" && user.Active {
		if err := ensureAnotherAdmin(ctx, tx, id); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM playback_progress WHERE profile_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM collections WHERE user_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getUser(ctx context.Context, query queryRower, id string) (User, error) {
	var user User
	var active int
	err := query.QueryRowContext(ctx, `SELECT id, username, role, active FROM users WHERE id = ?`, id).Scan(&user.ID, &user.Username, &user.Role, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	user.Active = active == 1
	return user, err
}

func ensureAnotherAdmin(ctx context.Context, query queryRower, excludedID string) error {
	var exists bool
	if err := query.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id != ? AND role = 'admin' AND active = 1)`, excludedID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrLastAdmin
	}
	return nil
}

func validateUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 64 {
		return "", ErrInvalidInput
	}
	for _, character := range username {
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '-' || character == '_' || character == '.') {
			return "", ErrInvalidInput
		}
	}
	return username, nil
}

func validPassword(password string) bool { return len(password) >= 12 && len(password) <= 256 }
func validRole(role string) bool         { return role == "admin" || role == "user" }

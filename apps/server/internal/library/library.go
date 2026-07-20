package library

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrConflict    = errors.New("a library already uses this path")
	ErrInvalidName = errors.New("library name is required")
	ErrInvalidPath = errors.New("library path must be an existing directory inside the media root")
	ErrNotFound    = errors.New("library not found")
)

type Library struct {
	ID                 string
	Name               string
	Path               string
	CreatedAt          time.Time
	LastScanAt         *time.Time
	LastScanDiscovered int
	LastScanIndexed    int
	LastScanUnchanged  int
	LastScanSkipped    int
}

type ScanSummary struct {
	FinishedAt time.Time
	Discovered int
	Indexed    int
	Unchanged  int
	Skipped    int
}

type Repository interface {
	List(ctx context.Context) ([]Library, error)
	Get(ctx context.Context, id string) (Library, error)
	Create(ctx context.Context, library Library) error
	PathExists(ctx context.Context, path string) (bool, error)
	Update(ctx context.Context, item Library) error
	Delete(ctx context.Context, id string) error
	RecordScan(ctx context.Context, id string, summary ScanSummary) error
}

type Service struct {
	repository Repository
	mediaRoot  string
}

func NewService(repository Repository, mediaRoot string) *Service {
	return &Service{repository: repository, mediaRoot: mediaRoot}
}

func (service *Service) List(ctx context.Context) ([]Library, error) {
	return service.repository.List(ctx)
}

func (service *Service) Get(ctx context.Context, id string) (Library, error) {
	return service.repository.Get(ctx, id)
}

func (service *Service) Update(ctx context.Context, id string, name string, path string) (Library, error) {
	current, err := service.repository.Get(ctx, id)
	if err != nil {
		return Library{}, err
	}
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return Library{}, ErrInvalidName
	}
	cleanPath, err := service.validatePath(path)
	if err != nil {
		return Library{}, err
	}
	if cleanPath != current.Path {
		exists, err := service.repository.PathExists(ctx, cleanPath)
		if err != nil {
			return Library{}, fmt.Errorf("check library path: %w", err)
		}
		if exists {
			return Library{}, ErrConflict
		}
	}
	current.Name = cleanName
	current.Path = cleanPath
	if err := service.repository.Update(ctx, current); err != nil {
		return Library{}, fmt.Errorf("update library: %w", err)
	}
	return current, nil
}

func (service *Service) Delete(ctx context.Context, id string) error {
	if _, err := service.repository.Get(ctx, id); err != nil {
		return err
	}
	return service.repository.Delete(ctx, id)
}

func (service *Service) RecordScan(ctx context.Context, id string, summary ScanSummary) error {
	return service.repository.RecordScan(ctx, id, summary)
}

func (service *Service) Add(ctx context.Context, name string, path string) (Library, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return Library{}, ErrInvalidName
	}

	cleanPath, err := service.validatePath(path)
	if err != nil {
		return Library{}, err
	}
	exists, err := service.repository.PathExists(ctx, cleanPath)
	if err != nil {
		return Library{}, fmt.Errorf("check library path: %w", err)
	}
	if exists {
		return Library{}, ErrConflict
	}

	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return Library{}, fmt.Errorf("generate library id: %w", err)
	}

	newLibrary := Library{
		ID:        hex.EncodeToString(idBytes),
		Name:      cleanName,
		Path:      cleanPath,
		CreatedAt: time.Now().UTC(),
	}
	if err := service.repository.Create(ctx, newLibrary); err != nil {
		return Library{}, fmt.Errorf("create library: %w", err)
	}
	return newLibrary, nil
}

func (service *Service) validatePath(path string) (string, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	root, err := filepath.Abs(service.mediaRoot)
	if err != nil {
		return "", ErrInvalidPath
	}
	absolutePath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", ErrInvalidPath
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", ErrInvalidPath
	}
	resolvedPath, err := filepath.EvalSymlinks(absolutePath)
	if err != nil {
		return "", ErrInvalidPath
	}
	relativePath, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", ErrInvalidPath
	}
	info, err := os.Stat(resolvedPath)
	if err != nil || !info.IsDir() {
		return "", ErrInvalidPath
	}
	return resolvedPath, nil
}

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (repository *SQLRepository) List(ctx context.Context) ([]Library, error) {
	rows, err := repository.db.QueryContext(ctx, `SELECT id, name, path, created_at, last_scan_at, last_scan_discovered, last_scan_indexed, last_scan_unchanged, last_scan_skipped FROM libraries ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("query libraries: %w", err)
	}
	defer rows.Close()

	libraries := make([]Library, 0)
	for rows.Next() {
		var item Library
		var createdAt string
		var lastScanAt sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &item.Path, &createdAt, &lastScanAt, &item.LastScanDiscovered, &item.LastScanIndexed, &item.LastScanUnchanged, &item.LastScanSkipped); err != nil {
			return nil, fmt.Errorf("scan library: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse library creation date: %w", err)
		}
		if lastScanAt.Valid {
			parsed, err := time.Parse(time.RFC3339Nano, lastScanAt.String)
			if err != nil {
				return nil, fmt.Errorf("parse library scan date: %w", err)
			}
			item.LastScanAt = &parsed
		}
		libraries = append(libraries, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate libraries: %w", err)
	}
	return libraries, nil
}

func (repository *SQLRepository) Get(ctx context.Context, id string) (Library, error) {
	var item Library
	var createdAt string
	var lastScanAt sql.NullString
	err := repository.db.QueryRowContext(ctx, `SELECT id, name, path, created_at, last_scan_at, last_scan_discovered, last_scan_indexed, last_scan_unchanged, last_scan_skipped FROM libraries WHERE id = ?`, id).
		Scan(&item.ID, &item.Name, &item.Path, &createdAt, &lastScanAt, &item.LastScanDiscovered, &item.LastScanIndexed, &item.LastScanUnchanged, &item.LastScanSkipped)
	if errors.Is(err, sql.ErrNoRows) {
		return Library{}, ErrNotFound
	}
	if err != nil {
		return Library{}, fmt.Errorf("query library: %w", err)
	}
	if lastScanAt.Valid {
		parsed, err := time.Parse(time.RFC3339Nano, lastScanAt.String)
		if err != nil {
			return Library{}, fmt.Errorf("parse library scan date: %w", err)
		}
		item.LastScanAt = &parsed
	}
	item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Library{}, fmt.Errorf("parse library creation date: %w", err)
	}
	return item, nil
}

func (repository *SQLRepository) Create(ctx context.Context, library Library) error {
	_, err := repository.db.ExecContext(ctx,
		`INSERT INTO libraries (id, name, path, created_at) VALUES (?, ?, ?, ?)`,
		library.ID, library.Name, library.Path, library.CreatedAt.Format(time.RFC3339Nano),
	)
	return err
}

func (repository *SQLRepository) PathExists(ctx context.Context, path string) (bool, error) {
	var exists bool
	err := repository.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM libraries WHERE path = ?)`, path).Scan(&exists)
	return exists, err
}

func (repository *SQLRepository) Update(ctx context.Context, item Library) error {
	result, err := repository.db.ExecContext(ctx, `UPDATE libraries SET name = ?, path = ? WHERE id = ?`, item.Name, item.Path, item.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrNotFound
	}
	return err
}

func (repository *SQLRepository) Delete(ctx context.Context, id string) error {
	_, err := repository.db.ExecContext(ctx, `DELETE FROM libraries WHERE id = ?`, id)
	return err
}

func (repository *SQLRepository) RecordScan(ctx context.Context, id string, summary ScanSummary) error {
	_, err := repository.db.ExecContext(ctx, `UPDATE libraries SET last_scan_at = ?, last_scan_discovered = ?, last_scan_indexed = ?, last_scan_unchanged = ?, last_scan_skipped = ? WHERE id = ?`,
		summary.FinishedAt.Format(time.RFC3339Nano), summary.Discovered, summary.Indexed, summary.Unchanged, summary.Skipped, id)
	return err
}

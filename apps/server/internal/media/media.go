package media

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"flex.local/server/internal/library"
)

var supportedExtensions = []string{
	".3gp", ".avi", ".flv", ".m2ts", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".mts", ".ogv", ".ts", ".webm", ".wmv",
}

var (
	ErrInvalidTitle = errors.New("media title is required")
	ErrNotFound     = errors.New("media not found")
)

type File struct {
	ID          string
	LibraryID   string
	Path        string
	Filename    string
	SizeBytes   int64
	DurationMS  int64
	Width       int
	Height      int
	Container   string
	VideoCodec  string
	AudioCodec  string
	ModifiedAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ProgressMS  int64
	Completed   bool
	Title       string
	Description string
	RecordedAt  *time.Time
	Favorite    bool
}

type MetadataInput struct {
	Title       string
	Description string
	RecordedAt  *time.Time
	Favorite    bool
}

type ScanResult struct {
	Discovered int
	Indexed    int
	Skipped    int
}

type Home struct {
	ContinueWatching []File
	RecentlyAdded    []File
}

type SearchResult struct {
	File
	LibraryName string
}

type Probe interface {
	Analyze(ctx context.Context, path string) (TechnicalMetadata, error)
}

type TechnicalMetadata struct {
	DurationMS int64
	Width      int
	Height     int
	Container  string
	VideoCodec string
	AudioCodec string
}

type LibrarySource interface {
	Get(ctx context.Context, id string) (library.Library, error)
	RecordScan(ctx context.Context, id string, summary library.ScanSummary) error
}

type Repository interface {
	List(ctx context.Context, libraryID string) ([]File, error)
	Favorites(ctx context.Context) ([]File, error)
	Get(ctx context.Context, id string) (File, error)
	Home(ctx context.Context, limit int) (Home, error)
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
	UpdateMetadata(ctx context.Context, id string, input MetadataInput) error
	Upsert(ctx context.Context, file File) error
	DeleteMissing(ctx context.Context, libraryID string, existingPaths []string) error
}

type Scanner struct {
	libraries  LibrarySource
	repository Repository
	probe      Probe
	logger     *slog.Logger
	thumbnails *ThumbnailGenerator
	transcoder *Transcoder
}

func NewScanner(libraries LibrarySource, repository Repository, probe Probe, thumbnails *ThumbnailGenerator, transcoder *Transcoder, logger *slog.Logger) *Scanner {
	return &Scanner{libraries: libraries, repository: repository, probe: probe, thumbnails: thumbnails, transcoder: transcoder, logger: logger}
}

func (scanner *Scanner) List(ctx context.Context, libraryID string) ([]File, error) {
	return scanner.repository.List(ctx, libraryID)
}

func (scanner *Scanner) Favorites(ctx context.Context) ([]File, error) {
	return scanner.repository.Favorites(ctx)
}

func (scanner *Scanner) Get(ctx context.Context, id string) (File, error) {
	return scanner.repository.Get(ctx, id)
}

func (scanner *Scanner) Home(ctx context.Context) (Home, error) {
	return scanner.repository.Home(ctx, 12)
}

func (scanner *Scanner) Search(ctx context.Context, query string) ([]SearchResult, error) {
	return scanner.repository.Search(ctx, strings.TrimSpace(query), 20)
}

func (scanner *Scanner) UpdateMetadata(ctx context.Context, id string, input MetadataInput) (File, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" || len(input.Title) > 200 {
		return File{}, ErrInvalidTitle
	}
	if len(input.Description) > 5_000 {
		return File{}, fmt.Errorf("description is too long")
	}
	if _, err := scanner.repository.Get(ctx, id); err != nil {
		return File{}, err
	}
	if err := scanner.repository.UpdateMetadata(ctx, id, input); err != nil {
		return File{}, fmt.Errorf("update media metadata: %w", err)
	}
	return scanner.repository.Get(ctx, id)
}

func (scanner *Scanner) Thumbnail(ctx context.Context, id string) (string, error) {
	item, err := scanner.repository.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return scanner.thumbnails.Generate(ctx, item)
}

func (scanner *Scanner) Transcode(ctx context.Context, id string) (string, error) {
	item, err := scanner.repository.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return scanner.transcoder.Generate(ctx, item)
}

func (scanner *Scanner) Scan(ctx context.Context, libraryID string) (ScanResult, error) {
	source, err := scanner.libraries.Get(ctx, libraryID)
	if err != nil {
		return ScanResult{}, err
	}

	result := ScanResult{}
	seenPaths := make([]string, 0)
	err = filepath.WalkDir(source.Path, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			scanner.logger.Warn("media path cannot be read", "path", path, "error", walkErr)
			result.Skipped++
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() || !entry.Type().IsRegular() || !isSupported(path) {
			return nil
		}

		result.Discovered++
		seenPaths = append(seenPaths, path)
		info, err := entry.Info()
		if err != nil {
			result.Skipped++
			return nil
		}
		technical, err := scanner.probe.Analyze(ctx, path)
		if err != nil {
			scanner.logger.Warn("media probe failed", "path", path, "error", err)
			result.Skipped++
			return nil
		}
		now := time.Now().UTC()
		id, err := randomID()
		if err != nil {
			return err
		}
		item := File{
			ID: id, LibraryID: libraryID, Path: path, Filename: filepath.Base(path),
			SizeBytes: info.Size(), DurationMS: technical.DurationMS, Width: technical.Width,
			Height: technical.Height, Container: technical.Container, VideoCodec: technical.VideoCodec,
			AudioCodec: technical.AudioCodec, ModifiedAt: info.ModTime().UTC(), CreatedAt: now, UpdatedAt: now,
		}
		if err := scanner.repository.Upsert(ctx, item); err != nil {
			return fmt.Errorf("index %s: %w", path, err)
		}
		result.Indexed++
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("walk library: %w", err)
	}
	if err := scanner.repository.DeleteMissing(ctx, libraryID, seenPaths); err != nil {
		return result, fmt.Errorf("remove missing media: %w", err)
	}
	if err := scanner.libraries.RecordScan(ctx, libraryID, library.ScanSummary{
		FinishedAt: time.Now().UTC(), Discovered: result.Discovered, Indexed: result.Indexed, Skipped: result.Skipped,
	}); err != nil {
		return result, fmt.Errorf("record scan result: %w", err)
	}
	return result, nil
}

func isSupported(path string) bool {
	return slices.Contains(supportedExtensions, strings.ToLower(filepath.Ext(path)))
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate media id: %w", err)
	}
	return hex.EncodeToString(value), nil
}

type SQLRepository struct{ db *sql.DB }

func NewSQLRepository(db *sql.DB) *SQLRepository { return &SQLRepository{db: db} }

func (repository *SQLRepository) List(ctx context.Context, libraryID string) ([]File, error) {
	return repository.queryFiles(ctx, `SELECT m.id, m.library_id, m.path, m.filename, m.size_bytes, m.duration_ms, m.width, m.height, m.container, m.video_codec, m.audio_codec, m.modified_at, m.created_at, m.updated_at, COALESCE(p.position_ms, 0), COALESCE(p.completed, 0), COALESCE(mm.title, ''), COALESCE(mm.description, ''), mm.recorded_at, COALESCE(mm.favorite, 0) FROM media_files m LEFT JOIN playback_progress p ON p.media_id = m.id AND p.profile_id = 'local' LEFT JOIN media_metadata mm ON mm.media_id = m.id WHERE m.library_id = ? ORDER BY COALESCE(NULLIF(mm.title, ''), m.filename) COLLATE NOCASE`, libraryID)
}

func (repository *SQLRepository) Favorites(ctx context.Context) ([]File, error) {
	return repository.queryFiles(ctx, `SELECT m.id, m.library_id, m.path, m.filename, m.size_bytes, m.duration_ms, m.width, m.height, m.container, m.video_codec, m.audio_codec, m.modified_at, m.created_at, m.updated_at, COALESCE(p.position_ms, 0), COALESCE(p.completed, 0), COALESCE(mm.title, ''), COALESCE(mm.description, ''), mm.recorded_at, COALESCE(mm.favorite, 0) FROM media_files m LEFT JOIN playback_progress p ON p.media_id = m.id AND p.profile_id = 'local' JOIN media_metadata mm ON mm.media_id = m.id WHERE mm.favorite = 1 ORDER BY COALESCE(NULLIF(mm.title, ''), m.filename) COLLATE NOCASE`)
}

func (repository *SQLRepository) Home(ctx context.Context, limit int) (Home, error) {
	continueWatching, err := repository.queryFiles(ctx, `SELECT m.id, m.library_id, m.path, m.filename, m.size_bytes, m.duration_ms, m.width, m.height, m.container, m.video_codec, m.audio_codec, m.modified_at, m.created_at, m.updated_at, p.position_ms, p.completed, COALESCE(mm.title, ''), COALESCE(mm.description, ''), mm.recorded_at, COALESCE(mm.favorite, 0) FROM media_files m JOIN playback_progress p ON p.media_id = m.id AND p.profile_id = 'local' LEFT JOIN media_metadata mm ON mm.media_id = m.id WHERE p.position_ms > 0 AND p.completed = 0 ORDER BY p.updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return Home{}, err
	}
	recentlyAdded, err := repository.queryFiles(ctx, `SELECT m.id, m.library_id, m.path, m.filename, m.size_bytes, m.duration_ms, m.width, m.height, m.container, m.video_codec, m.audio_codec, m.modified_at, m.created_at, m.updated_at, COALESCE(p.position_ms, 0), COALESCE(p.completed, 0), COALESCE(mm.title, ''), COALESCE(mm.description, ''), mm.recorded_at, COALESCE(mm.favorite, 0) FROM media_files m LEFT JOIN playback_progress p ON p.media_id = m.id AND p.profile_id = 'local' LEFT JOIN media_metadata mm ON mm.media_id = m.id ORDER BY m.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return Home{}, err
	}
	return Home{ContinueWatching: continueWatching, RecentlyAdded: recentlyAdded}, nil
}

func (repository *SQLRepository) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}
	pattern := "%" + escapeLike(query) + "%"
	rows, err := repository.db.QueryContext(ctx, `SELECT m.id, m.library_id, m.path, m.filename, m.size_bytes, m.duration_ms, m.width, m.height, m.container, m.video_codec, m.audio_codec, m.modified_at, m.created_at, m.updated_at, COALESCE(p.position_ms, 0), COALESCE(p.completed, 0), COALESCE(mm.title, ''), COALESCE(mm.description, ''), mm.recorded_at, COALESCE(mm.favorite, 0), l.name FROM media_files m JOIN libraries l ON l.id = m.library_id LEFT JOIN playback_progress p ON p.media_id = m.id AND p.profile_id = 'local' LEFT JOIN media_metadata mm ON mm.media_id = m.id WHERE COALESCE(NULLIF(mm.title, ''), m.filename) LIKE ? ESCAPE '\' ORDER BY COALESCE(NULLIF(mm.title, ''), m.filename) COLLATE NOCASE LIMIT ?`, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search media: %w", err)
	}
	defer rows.Close()
	results := make([]SearchResult, 0)
	for rows.Next() {
		var result SearchResult
		var completed, favorite int
		var modifiedAt, createdAt, updatedAt string
		var recordedAt sql.NullString
		err := rows.Scan(&result.ID, &result.LibraryID, &result.Path, &result.Filename, &result.SizeBytes, &result.DurationMS, &result.Width, &result.Height, &result.Container, &result.VideoCodec, &result.AudioCodec, &modifiedAt, &createdAt, &updatedAt, &result.ProgressMS, &completed, &result.Title, &result.Description, &recordedAt, &favorite, &result.LibraryName)
		if err != nil {
			return nil, fmt.Errorf("scan media search result: %w", err)
		}
		result.Completed = completed == 1
		result.Favorite = favorite == 1
		if err := applyMetadataDates(&result.File, recordedAt); err != nil {
			return nil, err
		}
		result.ModifiedAt, err = parseTime(modifiedAt)
		if err != nil {
			return nil, err
		}
		result.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		result.UpdatedAt, err = parseTime(updatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func (repository *SQLRepository) queryFiles(ctx context.Context, query string, arguments ...any) ([]File, error) {
	rows, err := repository.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("query media: %w", err)
	}
	defer rows.Close()
	items := make([]File, 0)
	for rows.Next() {
		var item File
		var completed, favorite int
		var modifiedAt, createdAt, updatedAt string
		var recordedAt sql.NullString
		if err := rows.Scan(&item.ID, &item.LibraryID, &item.Path, &item.Filename, &item.SizeBytes, &item.DurationMS, &item.Width, &item.Height, &item.Container, &item.VideoCodec, &item.AudioCodec, &modifiedAt, &createdAt, &updatedAt, &item.ProgressMS, &completed, &item.Title, &item.Description, &recordedAt, &favorite); err != nil {
			return nil, fmt.Errorf("scan media: %w", err)
		}
		item.Completed = completed == 1
		item.Favorite = favorite == 1
		if err := applyMetadataDates(&item, recordedAt); err != nil {
			return nil, err
		}
		item.ModifiedAt, err = parseTime(modifiedAt)
		if err != nil {
			return nil, err
		}
		item.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		item.UpdatedAt, err = parseTime(updatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repository *SQLRepository) Get(ctx context.Context, id string) (File, error) {
	var item File
	var modifiedAt, createdAt, updatedAt string
	var completed, favorite int
	var recordedAt sql.NullString
	err := repository.db.QueryRowContext(ctx, `SELECT m.id, m.library_id, m.path, m.filename, m.size_bytes, m.duration_ms, m.width, m.height, m.container, m.video_codec, m.audio_codec, m.modified_at, m.created_at, m.updated_at, COALESCE(p.position_ms, 0), COALESCE(p.completed, 0), COALESCE(mm.title, ''), COALESCE(mm.description, ''), mm.recorded_at, COALESCE(mm.favorite, 0) FROM media_files m LEFT JOIN playback_progress p ON p.media_id = m.id AND p.profile_id = 'local' LEFT JOIN media_metadata mm ON mm.media_id = m.id WHERE m.id = ?`, id).
		Scan(&item.ID, &item.LibraryID, &item.Path, &item.Filename, &item.SizeBytes, &item.DurationMS, &item.Width, &item.Height, &item.Container, &item.VideoCodec, &item.AudioCodec, &modifiedAt, &createdAt, &updatedAt, &item.ProgressMS, &completed, &item.Title, &item.Description, &recordedAt, &favorite)
	if errors.Is(err, sql.ErrNoRows) {
		return File{}, ErrNotFound
	}
	if err != nil {
		return File{}, fmt.Errorf("query media: %w", err)
	}
	item.Completed = completed == 1
	item.Favorite = favorite == 1
	if err := applyMetadataDates(&item, recordedAt); err != nil {
		return File{}, err
	}
	item.ModifiedAt, err = parseTime(modifiedAt)
	if err != nil {
		return File{}, err
	}
	item.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return File{}, err
	}
	item.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return File{}, err
	}
	return item, nil
}

func applyMetadataDates(item *File, recordedAt sql.NullString) error {
	if item.Title == "" {
		item.Title = strings.TrimSuffix(item.Filename, filepath.Ext(item.Filename))
	}
	if recordedAt.Valid {
		parsed, err := parseTime(recordedAt.String)
		if err != nil {
			return err
		}
		item.RecordedAt = &parsed
	}
	return nil
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse media date: %w", err)
	}
	return parsed, nil
}

func (repository *SQLRepository) Upsert(ctx context.Context, file File) error {
	_, err := repository.db.ExecContext(ctx, `INSERT INTO media_files (id, library_id, path, filename, size_bytes, duration_ms, width, height, container, video_codec, audio_codec, modified_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(path) DO UPDATE SET library_id=excluded.library_id, filename=excluded.filename, size_bytes=excluded.size_bytes, duration_ms=excluded.duration_ms, width=excluded.width, height=excluded.height, container=excluded.container, video_codec=excluded.video_codec, audio_codec=excluded.audio_codec, modified_at=excluded.modified_at, updated_at=excluded.updated_at`,
		file.ID, file.LibraryID, file.Path, file.Filename, file.SizeBytes, file.DurationMS, file.Width, file.Height, file.Container, file.VideoCodec, file.AudioCodec, file.ModifiedAt.Format(time.RFC3339Nano), file.CreatedAt.Format(time.RFC3339Nano), file.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (repository *SQLRepository) UpdateMetadata(ctx context.Context, id string, input MetadataInput) error {
	var recordedAt any
	if input.RecordedAt != nil {
		recordedAt = input.RecordedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := repository.db.ExecContext(ctx, `INSERT INTO media_metadata (media_id, title, description, recorded_at, favorite, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(media_id) DO UPDATE SET title=excluded.title, description=excluded.description, recorded_at=excluded.recorded_at, favorite=excluded.favorite, updated_at=excluded.updated_at`, id, input.Title, input.Description, recordedAt, input.Favorite, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (repository *SQLRepository) DeleteMissing(ctx context.Context, libraryID string, existingPaths []string) error {
	if len(existingPaths) == 0 {
		_, err := repository.db.ExecContext(ctx, `DELETE FROM media_files WHERE library_id = ?`, libraryID)
		return err
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(existingPaths)), ",")
	arguments := make([]any, 0, len(existingPaths)+1)
	arguments = append(arguments, libraryID)
	for _, path := range existingPaths {
		arguments = append(arguments, path)
	}
	_, err := repository.db.ExecContext(ctx, `DELETE FROM media_files WHERE library_id = ? AND path NOT IN (`+placeholders+`)`, arguments...)
	return err
}

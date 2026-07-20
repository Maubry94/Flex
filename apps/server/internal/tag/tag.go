package tag

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalid       = errors.New("invalid tag")
	ErrConflict      = errors.New("tag already exists")
	ErrMediaNotFound = errors.New("media not found")
	colorPattern     = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

type Tag struct {
	ID    string
	Name  string
	Color string
}

type Assignment struct {
	MediaID string
	Tag     Tag
}

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func (service *Service) List(ctx context.Context) ([]Tag, error) {
	rows, err := service.db.QueryContext(ctx, `SELECT id, name, color FROM tags ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()
	return scanTags(rows)
}

func (service *Service) Assignments(ctx context.Context) ([]Assignment, error) {
	rows, err := service.db.QueryContext(ctx, `SELECT mt.media_id, t.id, t.name, t.color FROM media_tags mt JOIN tags t ON t.id = mt.tag_id ORDER BY t.name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list tag assignments: %w", err)
	}
	defer rows.Close()
	items := make([]Assignment, 0)
	for rows.Next() {
		var item Assignment
		if err := rows.Scan(&item.MediaID, &item.Tag.ID, &item.Tag.Name, &item.Tag.Color); err != nil {
			return nil, fmt.Errorf("scan tag assignment: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag assignments: %w", err)
	}
	return items, nil
}

func (service *Service) Create(ctx context.Context, name string, color string) (Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 50 || !colorPattern.MatchString(color) {
		return Tag{}, ErrInvalid
	}
	id, err := randomID()
	if err != nil {
		return Tag{}, err
	}
	item := Tag{ID: id, Name: name, Color: strings.ToLower(color)}
	_, err = service.db.ExecContext(ctx, `INSERT INTO tags (id, name, color, created_at) VALUES (?, ?, ?, ?)`, item.ID, item.Name, item.Color, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Tag{}, ErrConflict
		}
		return Tag{}, fmt.Errorf("create tag: %w", err)
	}
	return item, nil
}

func (service *Service) ListForMedia(ctx context.Context, mediaID string) ([]Tag, error) {
	exists, err := service.mediaExists(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMediaNotFound
	}
	rows, err := service.db.QueryContext(ctx, `SELECT t.id, t.name, t.color FROM tags t JOIN media_tags mt ON mt.tag_id = t.id WHERE mt.media_id = ? ORDER BY t.name COLLATE NOCASE`, mediaID)
	if err != nil {
		return nil, fmt.Errorf("list media tags: %w", err)
	}
	defer rows.Close()
	return scanTags(rows)
}

func (service *Service) SetForMedia(ctx context.Context, mediaID string, tagIDs []string) ([]Tag, error) {
	if len(tagIDs) > 20 {
		return nil, ErrInvalid
	}
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin media tags update: %w", err)
	}
	defer transaction.Rollback()
	var exists bool
	if err := transaction.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_files WHERE id = ?)`, mediaID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check media: %w", err)
	}
	if !exists {
		return nil, ErrMediaNotFound
	}
	uniqueIDs := make(map[string]struct{}, len(tagIDs))
	for _, tagID := range tagIDs {
		if tagID == "" {
			return nil, ErrInvalid
		}
		if _, duplicate := uniqueIDs[tagID]; duplicate {
			continue
		}
		var tagExists bool
		if err := transaction.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tags WHERE id = ?)`, tagID).Scan(&tagExists); err != nil {
			return nil, fmt.Errorf("check tag: %w", err)
		}
		if !tagExists {
			return nil, ErrInvalid
		}
		uniqueIDs[tagID] = struct{}{}
	}
	if _, err := transaction.ExecContext(ctx, `DELETE FROM media_tags WHERE media_id = ?`, mediaID); err != nil {
		return nil, fmt.Errorf("clear media tags: %w", err)
	}
	for tagID := range uniqueIDs {
		if _, err := transaction.ExecContext(ctx, `INSERT INTO media_tags (media_id, tag_id) VALUES (?, ?)`, mediaID, tagID); err != nil {
			return nil, fmt.Errorf("assign media tag: %w", err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return nil, fmt.Errorf("commit media tags: %w", err)
	}
	return service.ListForMedia(ctx, mediaID)
}

func (service *Service) mediaExists(ctx context.Context, mediaID string) (bool, error) {
	var exists bool
	if err := service.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_files WHERE id = ?)`, mediaID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check media: %w", err)
	}
	return exists, nil
}

func scanTags(rows *sql.Rows) ([]Tag, error) {
	items := make([]Tag, 0)
	for rows.Next() {
		var item Tag
		if err := rows.Scan(&item.ID, &item.Name, &item.Color); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return items, nil
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate tag id: %w", err)
	}
	return hex.EncodeToString(value), nil
}

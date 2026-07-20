package collection

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid collection")
var ErrConflict = errors.New("collection already exists")
var ErrNotFound = errors.New("collection not found")
var ErrMediaNotFound = errors.New("media not found")

type Collection struct {
	ID         string
	Name       string
	MediaCount int
}
type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func (service *Service) List(ctx context.Context) ([]Collection, error) {
	rows, err := service.db.QueryContext(ctx, `SELECT c.id, c.name, COUNT(cm.media_id) FROM collections c LEFT JOIN collection_media cm ON cm.collection_id = c.id GROUP BY c.id ORDER BY c.name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()
	items := make([]Collection, 0)
	for rows.Next() {
		var item Collection
		if err := rows.Scan(&item.ID, &item.Name, &item.MediaCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (service *Service) Create(ctx context.Context, name string) (Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return Collection{}, ErrInvalid
	}
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return Collection{}, err
	}
	item := Collection{ID: hex.EncodeToString(value), Name: name}
	if _, err := service.db.ExecContext(ctx, `INSERT INTO collections (id, name, created_at) VALUES (?, ?, ?)`, item.ID, item.Name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Collection{}, ErrConflict
		}
		return Collection{}, fmt.Errorf("create collection: %w", err)
	}
	return item, nil
}

func (service *Service) Update(ctx context.Context, id string, name string) (Collection, error) {
	name = strings.TrimSpace(name)
	if id == "" || name == "" || len(name) > 100 {
		return Collection{}, ErrInvalid
	}
	result, err := service.db.ExecContext(ctx, `UPDATE collections SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Collection{}, ErrConflict
		}
		return Collection{}, fmt.Errorf("update collection: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil {
		return Collection{}, err
	} else if count == 0 {
		return Collection{}, ErrNotFound
	}
	var item Collection
	if err := service.db.QueryRowContext(ctx, `SELECT c.id, c.name, COUNT(cm.media_id) FROM collections c LEFT JOIN collection_media cm ON cm.collection_id = c.id WHERE c.id = ? GROUP BY c.id`, id).Scan(&item.ID, &item.Name, &item.MediaCount); err != nil {
		return Collection{}, fmt.Errorf("get updated collection: %w", err)
	}
	return item, nil
}

func (service *Service) Delete(ctx context.Context, id string) error {
	result, err := service.db.ExecContext(ctx, `DELETE FROM collections WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil {
		return err
	} else if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (service *Service) RemoveMedia(ctx context.Context, id string, mediaID string) error {
	result, err := service.db.ExecContext(ctx, `DELETE FROM collection_media WHERE collection_id = ? AND media_id = ?`, id, mediaID)
	if err != nil {
		return fmt.Errorf("remove media from collection: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil {
		return err
	} else if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (service *Service) ListForMedia(ctx context.Context, mediaID string) ([]Collection, error) {
	var exists bool
	if err := service.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_files WHERE id = ?)`, mediaID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMediaNotFound
	}
	rows, err := service.db.QueryContext(ctx, `SELECT c.id, c.name, (SELECT COUNT(*) FROM collection_media x WHERE x.collection_id = c.id) FROM collections c JOIN collection_media cm ON cm.collection_id = c.id WHERE cm.media_id = ? ORDER BY c.name COLLATE NOCASE`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Collection, 0)
	for rows.Next() {
		var item Collection
		if err := rows.Scan(&item.ID, &item.Name, &item.MediaCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (service *Service) SetForMedia(ctx context.Context, mediaID string, ids []string) ([]Collection, error) {
	if len(ids) > 50 {
		return nil, ErrInvalid
	}
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_files WHERE id = ?)`, mediaID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMediaNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM collection_media WHERE media_id = ?`, mediaID); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		var ok bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM collections WHERE id = ?)`, id).Scan(&ok); err != nil || !ok {
			return nil, ErrInvalid
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO collection_media (collection_id, media_id) VALUES (?, ?)`, id, mediaID); err != nil {
			return nil, err
		}
		seen[id] = true
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return service.ListForMedia(ctx, mediaID)
}

func (service *Service) MediaIDs(ctx context.Context, id string) ([]string, error) {
	rows, err := service.db.QueryContext(ctx, `SELECT media_id FROM collection_media WHERE collection_id = ? ORDER BY rowid`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]string, 0)
	for rows.Next() {
		var mediaID string
		if err := rows.Scan(&mediaID); err != nil {
			return nil, err
		}
		items = append(items, mediaID)
	}
	return items, rows.Err()
}

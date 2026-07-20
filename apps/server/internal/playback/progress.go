package playback

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Progress struct {
	MediaID    string
	PositionMS int64
	DurationMS int64
	Completed  bool
	UpdatedAt  time.Time
}

type Repository interface {
	Get(ctx context.Context, profileID string, mediaID string) (Progress, error)
	Save(ctx context.Context, profileID string, progress Progress) error
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (service *Service) Get(ctx context.Context, userID string, mediaID string) (Progress, error) {
	return service.repository.Get(ctx, userID, mediaID)
}

func (service *Service) Save(ctx context.Context, userID string, mediaID string, positionMS int64, durationMS int64) (Progress, error) {
	if positionMS < 0 {
		positionMS = 0
	}
	if durationMS < 0 {
		durationMS = 0
	}
	if durationMS > 0 && positionMS > durationMS {
		positionMS = durationMS
	}
	completed := durationMS > 0 && float64(positionMS)/float64(durationMS) >= 0.9
	progress := Progress{
		MediaID: mediaID, PositionMS: positionMS, DurationMS: durationMS,
		Completed: completed, UpdatedAt: time.Now().UTC(),
	}
	if err := service.repository.Save(ctx, userID, progress); err != nil {
		return Progress{}, fmt.Errorf("save playback progress: %w", err)
	}
	return progress, nil
}

type SQLRepository struct{ db *sql.DB }

func NewSQLRepository(db *sql.DB) *SQLRepository { return &SQLRepository{db: db} }

func (repository *SQLRepository) Get(ctx context.Context, profileID string, mediaID string) (Progress, error) {
	var progress Progress
	var completed int
	var updatedAt string
	err := repository.db.QueryRowContext(ctx, `SELECT media_id, position_ms, duration_ms, completed, updated_at FROM playback_progress WHERE profile_id = ? AND media_id = ?`, profileID, mediaID).
		Scan(&progress.MediaID, &progress.PositionMS, &progress.DurationMS, &completed, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Progress{MediaID: mediaID}, nil
	}
	if err != nil {
		return Progress{}, fmt.Errorf("query playback progress: %w", err)
	}
	progress.Completed = completed == 1
	progress.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Progress{}, fmt.Errorf("parse playback progress date: %w", err)
	}
	return progress, nil
}

func (repository *SQLRepository) Save(ctx context.Context, profileID string, progress Progress) error {
	_, err := repository.db.ExecContext(ctx, `INSERT INTO playback_progress (profile_id, media_id, position_ms, duration_ms, completed, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(profile_id, media_id) DO UPDATE SET position_ms=excluded.position_ms, duration_ms=excluded.duration_ms, completed=excluded.completed, updated_at=excluded.updated_at`,
		profileID, progress.MediaID, progress.PositionMS, progress.DurationMS, progress.Completed, progress.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

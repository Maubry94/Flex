package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func Open(configPath string) (*sql.DB, error) {
	if err := os.MkdirAll(configPath, 0o750); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	databasePath := filepath.Join(configPath, "flex.db")
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := migrate(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	migrations := []string{
		`
		CREATE TABLE IF NOT EXISTS libraries (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS media_files (
			id TEXT PRIMARY KEY,
			library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
			path TEXT NOT NULL UNIQUE,
			filename TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			container TEXT NOT NULL,
			video_codec TEXT NOT NULL,
			audio_codec TEXT NOT NULL,
			modified_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_media_files_library_id ON media_files(library_id);
		CREATE TABLE IF NOT EXISTS playback_progress (
			profile_id TEXT NOT NULL,
			media_id TEXT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
			position_ms INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL,
			completed INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (profile_id, media_id)
		);
		CREATE INDEX IF NOT EXISTS idx_playback_progress_updated_at ON playback_progress(profile_id, updated_at DESC);
		`,
		`
		ALTER TABLE libraries ADD COLUMN last_scan_at TEXT;
		ALTER TABLE libraries ADD COLUMN last_scan_discovered INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE libraries ADD COLUMN last_scan_indexed INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE libraries ADD COLUMN last_scan_skipped INTEGER NOT NULL DEFAULT 0;
		`,
		`
		CREATE TABLE media_metadata (
			media_id TEXT PRIMARY KEY REFERENCES media_files(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			recorded_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL
		);
		`,
	}

	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON;`); err != nil {
		return fmt.Errorf("configure database: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	for index, migration := range migrations {
		version := index + 1
		var applied bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if applied {
			continue
		}
		transaction, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version, err)
		}
		if _, err := transaction.ExecContext(ctx, migration); err != nil {
			_ = transaction.Rollback()
			return fmt.Errorf("apply migration %d: %w", version, err)
		}
		if _, err := transaction.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			_ = transaction.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}
		if err := transaction.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}
	return nil
}

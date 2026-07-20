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
		`
		CREATE TABLE tags (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL COLLATE NOCASE UNIQUE,
			color TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE media_tags (
			media_id TEXT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
			tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
			PRIMARY KEY (media_id, tag_id)
		);
		CREATE INDEX idx_media_tags_tag_id ON media_tags(tag_id);
		`,
		`
		CREATE TABLE collections (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL COLLATE NOCASE UNIQUE,
			created_at TEXT NOT NULL
		);
		CREATE TABLE collection_media (
			collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
			media_id TEXT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
			PRIMARY KEY (collection_id, media_id)
		);
		CREATE INDEX idx_collection_media_media_id ON collection_media(media_id);
		`,
		`
		ALTER TABLE libraries ADD COLUMN last_scan_unchanged INTEGER NOT NULL DEFAULT 0;
		`,
		`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL COLLATE NOCASE UNIQUE,
			display_name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('admin', 'user')),
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE sessions (
			token_hash TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL
		);
		CREATE INDEX idx_sessions_user_id ON sessions(user_id);
		CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
		`,
		`
		CREATE TABLE user_media_state (
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			media_id TEXT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
			favorite INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (user_id, media_id)
		);
		CREATE INDEX idx_user_media_state_favorites ON user_media_state(user_id, favorite);
		INSERT INTO user_media_state (user_id, media_id, favorite, updated_at)
		SELECT (SELECT id FROM users ORDER BY created_at LIMIT 1), media_id, favorite, updated_at
		FROM media_metadata
		WHERE favorite = 1 AND EXISTS (SELECT 1 FROM users);
		INSERT OR IGNORE INTO playback_progress (profile_id, media_id, position_ms, duration_ms, completed, updated_at)
		SELECT (SELECT id FROM users ORDER BY created_at LIMIT 1), media_id, position_ms, duration_ms, completed, updated_at
		FROM playback_progress
		WHERE profile_id = 'local' AND EXISTS (SELECT 1 FROM users);
		`,
		`
		CREATE TABLE collections_v2 (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL COLLATE NOCASE,
			created_at TEXT NOT NULL,
			UNIQUE (user_id, name)
		);
		INSERT INTO collections_v2 (id, user_id, name, created_at)
		SELECT id, COALESCE((SELECT id FROM users ORDER BY created_at LIMIT 1), 'local'), name, created_at FROM collections;
		CREATE TABLE collection_media_v2 (
			collection_id TEXT NOT NULL REFERENCES collections_v2(id) ON DELETE CASCADE,
			media_id TEXT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
			PRIMARY KEY (collection_id, media_id)
		);
		INSERT INTO collection_media_v2 (collection_id, media_id) SELECT collection_id, media_id FROM collection_media;
		DROP TABLE collection_media;
		DROP TABLE collections;
		ALTER TABLE collections_v2 RENAME TO collections;
		ALTER TABLE collection_media_v2 RENAME TO collection_media;
		CREATE INDEX idx_collections_user_id ON collections(user_id);
		CREATE INDEX idx_collection_media_media_id ON collection_media(media_id);
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

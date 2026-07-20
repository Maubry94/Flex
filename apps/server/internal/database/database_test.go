package database

import (
	"context"
	"testing"
)

func TestOpenCreatesDatabaseAndSchema(t *testing.T) {
	directory := t.TempDir()
	db, err := Open(directory)
	if err != nil {
		t.Fatalf("Open() returned an error: %v", err)
	}

	var tableName string
	err = db.QueryRowContext(context.Background(), `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'libraries'`).Scan(&tableName)
	if err != nil {
		t.Fatalf("libraries schema is missing: %v", err)
	}
	if tableName != "libraries" {
		t.Fatalf("unexpected table: %s", tableName)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}
	db, err = Open(directory)
	if err != nil {
		t.Fatalf("reopening an already migrated database failed: %v", err)
	}
	defer db.Close()
}

func TestOpenMigratesExistingDatabaseWithoutLosingLibraries(t *testing.T) {
	directory := t.TempDir()
	db, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name, path, created_at) VALUES ('library-1', 'Vidéos', '/media/videos', '2026-07-20T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE libraries DROP COLUMN last_scan_unchanged`); err != nil {
		t.Fatalf("prepare previous schema: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM schema_migrations WHERE version = (SELECT MAX(version) FROM schema_migrations)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	migrated, err := Open(directory)
	if err != nil {
		t.Fatalf("migrate previous database: %v", err)
	}
	defer migrated.Close()
	var name string
	var unchanged int
	if err := migrated.QueryRow(`SELECT name, last_scan_unchanged FROM libraries WHERE id = 'library-1'`).Scan(&name, &unchanged); err != nil {
		t.Fatalf("read migrated library: %v", err)
	}
	if name != "Vidéos" || unchanged != 0 {
		t.Fatalf("unexpected migrated library: name=%q unchanged=%d", name, unchanged)
	}
}

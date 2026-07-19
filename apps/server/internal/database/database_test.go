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

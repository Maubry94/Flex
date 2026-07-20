package collection_test

import (
	"context"
	"testing"
	"time"

	"flex.local/server/internal/collection"
	"flex.local/server/internal/database"
)

func TestCreateAndAssignCollection(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO libraries (id, name, path, created_at) VALUES ('library-1', 'Vidéos', '/media', ?)`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO media_files (id, library_id, path, filename, size_bytes, duration_ms, width, height, container, video_codec, audio_codec, modified_at, created_at, updated_at) VALUES ('media-1', 'library-1', '/media/video.mp4', 'video.mp4', 1, 1000, 1920, 1080, 'mp4', 'h264', 'aac', ?, ?, ?)`, now, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, username, display_name, password_hash, role, active, created_at, updated_at) VALUES ('user-1', 'alice', 'alice', 'hash', 'admin', 1, ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}
	service := collection.NewService(db)
	created, err := service.Create(context.Background(), "user-1", "  Voyages  ")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	assigned, err := service.SetForMedia(context.Background(), "user-1", "media-1", []string{created.ID})
	if err != nil {
		t.Fatalf("assign collection: %v", err)
	}
	if len(assigned) != 1 || assigned[0].ID != created.ID {
		t.Fatalf("unexpected assignments: %#v", assigned)
	}
	mediaIDs, err := service.MediaIDs(context.Background(), "user-1", created.ID)
	if err != nil || len(mediaIDs) != 1 || mediaIDs[0] != "media-1" {
		t.Fatalf("unexpected media ids: %#v, %v", mediaIDs, err)
	}
	updated, err := service.Update(context.Background(), "user-1", created.ID, "Escapades")
	if err != nil || updated.Name != "Escapades" || updated.MediaCount != 1 {
		t.Fatalf("unexpected updated collection: %#v, %v", updated, err)
	}
	if err := service.RemoveMedia(context.Background(), "user-1", created.ID, "media-1"); err != nil {
		t.Fatalf("remove media: %v", err)
	}
	mediaIDs, err = service.MediaIDs(context.Background(), "user-1", created.ID)
	if err != nil || len(mediaIDs) != 0 {
		t.Fatalf("collection should be empty: %#v, %v", mediaIDs, err)
	}
	if err := service.Delete(context.Background(), "user-1", created.ID); err != nil {
		t.Fatalf("delete collection: %v", err)
	}
	if _, err := service.Update(context.Background(), "user-1", created.ID, "Introuvable"); err != collection.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := collection.NewService(db).Create(context.Background(), "user-1", " "); err != collection.ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestCollectionsAreIsolatedByUser(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO users (id, username, display_name, password_hash, role, active, created_at, updated_at) VALUES ('alice', 'alice', 'alice', 'hash', 'admin', 1, ?, ?), ('bob', 'bob', 'bob', 'hash', 'user', 1, ?, ?)`, now, now, now, now); err != nil {
		t.Fatal(err)
	}
	service := collection.NewService(db)
	aliceCollection, err := service.Create(context.Background(), "alice", "Vacances")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), "bob", "Vacances"); err != nil {
		t.Fatalf("same name should be available to another user: %v", err)
	}
	aliceItems, err := service.List(context.Background(), "alice")
	if err != nil || len(aliceItems) != 1 || aliceItems[0].ID != aliceCollection.ID {
		t.Fatalf("unexpected Alice collections: %#v, %v", aliceItems, err)
	}
	if err := service.Delete(context.Background(), "bob", aliceCollection.ID); err != collection.ErrNotFound {
		t.Fatalf("Bob should not delete Alice's collection: %v", err)
	}
}

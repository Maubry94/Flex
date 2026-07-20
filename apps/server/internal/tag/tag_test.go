package tag_test

import (
	"context"
	"testing"
	"time"

	"flex.local/server/internal/database"
	"flex.local/server/internal/tag"
)

func TestCreateAndAssignTags(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO libraries (id, name, path, created_at) VALUES ('library-1', 'Vidéos', '/media', ?)`, now); err != nil {
		t.Fatalf("insert library: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO media_files (id, library_id, path, filename, size_bytes, duration_ms, width, height, container, video_codec, audio_codec, modified_at, created_at, updated_at) VALUES ('media-1', 'library-1', '/media/video.mp4', 'video.mp4', 1, 1000, 1920, 1080, 'mp4', 'h264', 'aac', ?, ?, ?)`, now, now, now); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	service := tag.NewService(db)
	created, err := service.Create(context.Background(), "  Voyage  ", "#7C3AED")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if created.Name != "Voyage" || created.Color != "#7c3aed" {
		t.Fatalf("unexpected created tag: %#v", created)
	}
	assigned, err := service.SetForMedia(context.Background(), "media-1", []string{created.ID})
	if err != nil {
		t.Fatalf("assign tag: %v", err)
	}
	if len(assigned) != 1 || assigned[0] != created {
		t.Fatalf("unexpected assigned tags: %#v", assigned)
	}
	assignments, err := service.Assignments(context.Background())
	if err != nil {
		t.Fatalf("list assignments: %v", err)
	}
	if len(assignments) != 1 || assignments[0].MediaID != "media-1" || assignments[0].Tag != created {
		t.Fatalf("unexpected assignments: %#v", assignments)
	}
}

func TestCreateRejectsInvalidTag(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	service := tag.NewService(db)
	if _, err := service.Create(context.Background(), "", "purple"); err != tag.ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

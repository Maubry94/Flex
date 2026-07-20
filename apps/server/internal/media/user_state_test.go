package media

import (
	"context"
	"testing"

	"flex.local/server/internal/database"
)

func TestSQLRepositoryIsolatesMediaStateByUser(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	statements := []string{
		`INSERT INTO users (id, username, display_name, password_hash, role, active, created_at, updated_at) VALUES ('alice', 'alice', 'alice', 'hash', 'admin', 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'), ('bob', 'bob', 'bob', 'hash', 'user', 1, '2026-01-01T00:00:01Z', '2026-01-01T00:00:01Z')`,
		`INSERT INTO libraries (id, name, path, created_at) VALUES ('library', 'Vidéos', '/media', '2026-01-01T00:00:00Z')`,
		`INSERT INTO media_files (id, library_id, path, filename, size_bytes, duration_ms, width, height, container, video_codec, audio_codec, modified_at, created_at, updated_at) VALUES ('video', 'library', '/media/video.mp4', 'video.mp4', 1, 100000, 1920, 1080, 'mp4', 'h264', 'aac', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO playback_progress (profile_id, media_id, position_ms, duration_ms, completed, updated_at) VALUES ('alice', 'video', 50000, 100000, 0, '2026-01-01T00:00:00Z')`,
		`INSERT INTO user_media_state (user_id, media_id, favorite, updated_at) VALUES ('alice', 'video', 1, '2026-01-01T00:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	repository := NewSQLRepository(db)
	alice, err := repository.Get(ctx, "alice", "video")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := repository.Get(ctx, "bob", "video")
	if err != nil {
		t.Fatal(err)
	}
	if alice.ProgressMS != 50000 || !alice.Favorite {
		t.Fatalf("unexpected Alice state: %#v", alice)
	}
	if bob.ProgressMS != 0 || bob.Favorite {
		t.Fatalf("Alice state leaked to Bob: %#v", bob)
	}
}

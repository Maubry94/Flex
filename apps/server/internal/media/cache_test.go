package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestThumbnailRemoveDeletesOnlyRequestedMedia(t *testing.T) {
	generator := NewThumbnailGenerator(t.TempDir(), "ffmpeg")
	if err := os.MkdirAll(generator.cachePath, 0o750); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(generator.cachePath, "media-1.jpg")
	other := filepath.Join(generator.cachePath, "media-2.jpg")
	for _, path := range []string{target, other} {
		if err := os.WriteFile(path, []byte("thumbnail"), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	if err := generator.Remove("media-1"); err != nil {
		t.Fatalf("Remove() returned an error: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("requested thumbnail still exists: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("unrelated thumbnail was removed: %v", err)
	}
}

func TestTranscoderRemoveDeletesAllMediaVersions(t *testing.T) {
	transcoder := NewTranscoder(t.TempDir(), "ffmpeg")
	for _, name := range []string{"media-1-100", "media-1-200", "media-2-100"} {
		if err := os.MkdirAll(filepath.Join(transcoder.cachePath, name), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if err := transcoder.Remove("media-1"); err != nil {
		t.Fatalf("Remove() returned an error: %v", err)
	}
	for _, name := range []string{"media-1-100", "media-1-200"} {
		if _, err := os.Stat(filepath.Join(transcoder.cachePath, name)); !os.IsNotExist(err) {
			t.Fatalf("stale transcode %s still exists: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(transcoder.cachePath, "media-2-100")); err != nil {
		t.Fatalf("unrelated transcode was removed: %v", err)
	}
}

func TestCacheRemovalRejectsUnsafeMediaID(t *testing.T) {
	if err := NewThumbnailGenerator(t.TempDir(), "ffmpeg").Remove("../media"); err == nil {
		t.Fatal("expected unsafe thumbnail media id to be rejected")
	}
	if err := NewTranscoder(t.TempDir(), "ffmpeg").Remove("../media"); err == nil {
		t.Fatal("expected unsafe transcode media id to be rejected")
	}
}

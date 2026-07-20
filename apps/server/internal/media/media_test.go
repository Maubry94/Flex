package media

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"flex.local/server/internal/library"
)

type fakeLibrarySource struct{ item library.Library }

func (source fakeLibrarySource) Get(context.Context, string) (library.Library, error) {
	return source.item, nil
}

func (source fakeLibrarySource) RecordScan(context.Context, string, library.ScanSummary) error {
	return nil
}

type fakeProbe struct {
	metadata TechnicalMetadata
	calls    int
	err      error
}

func (probe *fakeProbe) Analyze(context.Context, string) (TechnicalMetadata, error) {
	probe.calls++
	return probe.metadata, probe.err
}

type memoryMediaRepository struct{ items []File }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (repository *memoryMediaRepository) List(context.Context, string) ([]File, error) {
	return repository.items, nil
}

func (repository *memoryMediaRepository) Favorites(context.Context) ([]File, error) {
	return repository.items, nil
}

func (repository *memoryMediaRepository) Get(_ context.Context, id string) (File, error) {
	for _, item := range repository.items {
		if item.ID == id {
			return item, nil
		}
	}
	return File{}, ErrNotFound
}

func (repository *memoryMediaRepository) Home(context.Context, int) (Home, error) {
	return Home{RecentlyAdded: repository.items}, nil
}

func (repository *memoryMediaRepository) Search(context.Context, string, int) ([]SearchResult, error) {
	return nil, nil
}
func (repository *memoryMediaRepository) Folders(context.Context, string) ([]FolderAssignment, error) {
	return nil, nil
}

func (repository *memoryMediaRepository) UpdateMetadata(_ context.Context, id string, input MetadataInput) error {
	for index := range repository.items {
		if repository.items[index].ID == id {
			repository.items[index].Title = input.Title
			repository.items[index].Description = input.Description
			repository.items[index].RecordedAt = input.RecordedAt
			repository.items[index].Favorite = input.Favorite
			return nil
		}
	}
	return ErrNotFound
}

func (repository *memoryMediaRepository) Upsert(_ context.Context, item File) error {
	for index := range repository.items {
		if repository.items[index].Path == item.Path {
			repository.items[index] = item
			return nil
		}
	}
	repository.items = append(repository.items, item)
	return nil
}

func (repository *memoryMediaRepository) DeleteMissing(_ context.Context, _ string, existingPaths []string) (int, error) {
	seen := make(map[string]bool, len(existingPaths))
	for _, path := range existingPaths {
		seen[path] = true
	}
	kept := repository.items[:0]
	removed := 0
	for _, item := range repository.items {
		if seen[item.Path] {
			kept = append(kept, item)
		} else {
			removed++
		}
	}
	repository.items = kept
	return removed, nil
}

func TestScanIndexesSupportedVideo(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "video.mov")
	if err := os.WriteFile(videoPath, []byte("test video"), 0o640); err != nil {
		t.Fatalf("create test video: %v", err)
	}
	repository := &memoryMediaRepository{}
	probe := &fakeProbe{metadata: TechnicalMetadata{DurationMS: 42_000, Width: 1920, Height: 1080, VideoCodec: "h264"}}
	scanner := NewScanner(
		fakeLibrarySource{item: library.Library{ID: "library", Path: root}},
		repository,
		probe,
		nil,
		nil,
		discardLogger(),
	)

	result, err := scanner.Scan(context.Background(), "library")
	if err != nil {
		t.Fatalf("Scan() returned an error: %v", err)
	}
	if result.Discovered != 1 || result.Indexed != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected scan result: %#v", result)
	}
	if len(repository.items) != 1 || repository.items[0].Filename != "video.mov" {
		t.Fatalf("unexpected indexed media: %#v", repository.items)
	}

	secondResult, err := scanner.Scan(context.Background(), "library")
	if err != nil {
		t.Fatalf("second Scan() returned an error: %v", err)
	}
	if secondResult.Indexed != 0 || secondResult.Unchanged != 1 || probe.calls != 1 {
		t.Fatalf("unchanged file should not be probed again: result=%#v calls=%d", secondResult, probe.calls)
	}
}

func TestScanReportsUnreadableVideo(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "broken.mov")
	if err := os.WriteFile(videoPath, []byte("not a video"), 0o640); err != nil {
		t.Fatal(err)
	}
	scanner := NewScanner(
		fakeLibrarySource{item: library.Library{ID: "library", Path: root}},
		&memoryMediaRepository{},
		&fakeProbe{err: errors.New("invalid data")},
		nil, nil, discardLogger(),
	)

	result, err := scanner.Scan(context.Background(), "library")
	if err != nil {
		t.Fatalf("Scan() returned an error: %v", err)
	}
	if result.Skipped != 1 || len(result.Issues) != 1 || result.Issues[0].Filename != "broken.mov" {
		t.Fatalf("unexpected issues: %#v", result)
	}
}

func TestScanRemovesMissingVideo(t *testing.T) {
	root := t.TempDir()
	repository := &memoryMediaRepository{items: []File{{ID: "missing", LibraryID: "library", Path: filepath.Join(root, "missing.mp4")}}}
	cacheRoot := t.TempDir()
	thumbnails := NewThumbnailGenerator(cacheRoot, "ffmpeg")
	transcoder := NewTranscoder(cacheRoot, "ffmpeg")
	if err := os.MkdirAll(thumbnails.cachePath, 0o750); err != nil {
		t.Fatal(err)
	}
	thumbnailPath := filepath.Join(thumbnails.cachePath, "missing.jpg")
	if err := os.WriteFile(thumbnailPath, []byte("thumbnail"), 0o640); err != nil {
		t.Fatal(err)
	}
	transcodePath := filepath.Join(transcoder.cachePath, "missing-100")
	if err := os.MkdirAll(transcodePath, 0o750); err != nil {
		t.Fatal(err)
	}
	scanner := NewScanner(fakeLibrarySource{item: library.Library{ID: "library", Path: root}}, repository, &fakeProbe{}, thumbnails, transcoder, discardLogger())

	result, err := scanner.Scan(context.Background(), "library")
	if err != nil {
		t.Fatalf("Scan() returned an error: %v", err)
	}
	if result.Removed != 1 || len(repository.items) != 0 {
		t.Fatalf("missing media was not removed: result=%#v items=%#v", result, repository.items)
	}
	if _, err := os.Stat(thumbnailPath); !os.IsNotExist(err) {
		t.Fatalf("stale thumbnail still exists: %v", err)
	}
	if _, err := os.Stat(transcodePath); !os.IsNotExist(err) {
		t.Fatalf("stale transcode still exists: %v", err)
	}
}

func TestScanPreservesIndexWhenDirectoryWalkIsIncomplete(t *testing.T) {
	root := t.TempDir()
	indexed := File{ID: "preserved", LibraryID: "library", Path: filepath.Join(root, "temporarily-unavailable", "video.mp4")}
	repository := &memoryMediaRepository{items: []File{indexed}}
	scanner := NewScanner(fakeLibrarySource{item: library.Library{ID: "library", Path: root}}, repository, &fakeProbe{}, nil, nil, discardLogger())
	scanner.walkDir = func(_ string, visit fs.WalkDirFunc) error {
		return visit(filepath.Join(root, "temporarily-unavailable"), nil, errors.New("permission denied"))
	}

	result, err := scanner.Scan(context.Background(), "library")
	if err != nil {
		t.Fatalf("Scan() returned an error: %v", err)
	}
	if result.Skipped != 1 || len(result.Issues) != 1 || result.Removed != 0 {
		t.Fatalf("unexpected incomplete scan result: %#v", result)
	}
	if len(repository.items) != 1 || repository.items[0].ID != indexed.ID {
		t.Fatalf("indexed media was removed after an incomplete walk: %#v", repository.items)
	}
}

func TestUpdateMetadata(t *testing.T) {
	repository := &memoryMediaRepository{items: []File{{ID: "media-1", Title: "Ancien titre"}}}
	scanner := NewScanner(fakeLibrarySource{}, repository, &fakeProbe{}, nil, nil, discardLogger())

	updated, err := scanner.UpdateMetadata(context.Background(), "media-1", MetadataInput{
		Title: "  Nouveau titre  ", Description: "  Description  ", Favorite: true,
	})
	if err != nil {
		t.Fatalf("UpdateMetadata() returned an error: %v", err)
	}
	if updated.Title != "Nouveau titre" || updated.Description != "Description" || !updated.Favorite {
		t.Fatalf("unexpected updated metadata: %#v", updated)
	}
}

func TestUpdateMetadataRejectsEmptyTitle(t *testing.T) {
	repository := &memoryMediaRepository{items: []File{{ID: "media-1", Title: "Titre"}}}
	scanner := NewScanner(fakeLibrarySource{}, repository, &fakeProbe{}, nil, nil, discardLogger())

	if _, err := scanner.UpdateMetadata(context.Background(), "media-1", MetadataInput{Title: "  "}); err != ErrInvalidTitle {
		t.Fatalf("expected ErrInvalidTitle, got %v", err)
	}
}

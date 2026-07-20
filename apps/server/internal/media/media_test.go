package media

import (
	"context"
	"io"
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

type fakeProbe struct{ metadata TechnicalMetadata }

func (probe fakeProbe) Analyze(context.Context, string) (TechnicalMetadata, error) {
	return probe.metadata, nil
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
	repository.items = append(repository.items, item)
	return nil
}

func (repository *memoryMediaRepository) DeleteMissing(context.Context, string, []string) error {
	return nil
}

func TestScanIndexesSupportedVideo(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "video.mov")
	if err := os.WriteFile(videoPath, []byte("test video"), 0o640); err != nil {
		t.Fatalf("create test video: %v", err)
	}
	repository := &memoryMediaRepository{}
	scanner := NewScanner(
		fakeLibrarySource{item: library.Library{ID: "library", Path: root}},
		repository,
		fakeProbe{metadata: TechnicalMetadata{DurationMS: 42_000, Width: 1920, Height: 1080, VideoCodec: "h264"}},
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
}

func TestUpdateMetadata(t *testing.T) {
	repository := &memoryMediaRepository{items: []File{{ID: "media-1", Title: "Ancien titre"}}}
	scanner := NewScanner(fakeLibrarySource{}, repository, fakeProbe{}, nil, nil, discardLogger())

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
	scanner := NewScanner(fakeLibrarySource{}, repository, fakeProbe{}, nil, nil, discardLogger())

	if _, err := scanner.UpdateMetadata(context.Background(), "media-1", MetadataInput{Title: "  "}); err != ErrInvalidTitle {
		t.Fatalf("expected ErrInvalidTitle, got %v", err)
	}
}

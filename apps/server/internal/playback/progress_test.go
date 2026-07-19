package playback

import (
	"context"
	"testing"
)

type memoryRepository struct{ progress Progress }

func (repository *memoryRepository) Get(context.Context, string, string) (Progress, error) {
	return repository.progress, nil
}

func (repository *memoryRepository) Save(_ context.Context, _ string, progress Progress) error {
	repository.progress = progress
	return nil
}

func TestSaveMarksVideoCompletedAtNinetyPercent(t *testing.T) {
	repository := &memoryRepository{}
	service := NewService(repository)

	progress, err := service.Save(context.Background(), "media", 90_000, 100_000)
	if err != nil {
		t.Fatalf("Save() returned an error: %v", err)
	}
	if !progress.Completed {
		t.Fatal("progress should be completed at 90 percent")
	}
}

func TestSaveKeepsVideoInProgressBeforeThreshold(t *testing.T) {
	service := NewService(&memoryRepository{})
	progress, err := service.Save(context.Background(), "media", 50_000, 100_000)
	if err != nil {
		t.Fatalf("Save() returned an error: %v", err)
	}
	if progress.Completed {
		t.Fatal("progress should not be completed before 90 percent")
	}
}

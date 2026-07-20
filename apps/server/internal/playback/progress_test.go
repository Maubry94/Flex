package playback

import (
	"context"
	"testing"
)

type memoryRepository struct {
	progress Progress
	userID   string
}

func (repository *memoryRepository) Get(context.Context, string, string) (Progress, error) {
	return repository.progress, nil
}

func (repository *memoryRepository) Save(_ context.Context, userID string, progress Progress) error {
	repository.userID = userID
	repository.progress = progress
	return nil
}

func TestSaveMarksVideoCompletedAtNinetyPercent(t *testing.T) {
	repository := &memoryRepository{}
	service := NewService(repository)

	progress, err := service.Save(context.Background(), "user", "media", 90_000, 100_000)
	if err != nil {
		t.Fatalf("Save() returned an error: %v", err)
	}
	if !progress.Completed {
		t.Fatal("progress should be completed at 90 percent")
	}
	if repository.userID != "user" {
		t.Fatalf("progress stored for %q instead of the authenticated user", repository.userID)
	}
}

func TestSaveKeepsVideoInProgressBeforeThreshold(t *testing.T) {
	service := NewService(&memoryRepository{})
	progress, err := service.Save(context.Background(), "user", "media", 50_000, 100_000)
	if err != nil {
		t.Fatalf("Save() returned an error: %v", err)
	}
	if progress.Completed {
		t.Fatal("progress should not be completed before 90 percent")
	}
}

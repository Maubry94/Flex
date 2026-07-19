package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type memoryRepository struct {
	items []Library
}

func (repository *memoryRepository) List(context.Context) ([]Library, error) {
	return repository.items, nil
}

func (repository *memoryRepository) Get(_ context.Context, id string) (Library, error) {
	for _, item := range repository.items {
		if item.ID == id {
			return item, nil
		}
	}
	return Library{}, ErrNotFound
}

func (repository *memoryRepository) Create(_ context.Context, item Library) error {
	repository.items = append(repository.items, item)
	return nil
}

func (repository *memoryRepository) PathExists(_ context.Context, path string) (bool, error) {
	for _, item := range repository.items {
		if item.Path == path {
			return true, nil
		}
	}
	return false, nil
}

func (repository *memoryRepository) Update(_ context.Context, updated Library) error {
	for index, item := range repository.items {
		if item.ID == updated.ID {
			repository.items[index] = updated
			return nil
		}
	}
	return ErrNotFound
}

func (repository *memoryRepository) Delete(_ context.Context, id string) error {
	for index, item := range repository.items {
		if item.ID == id {
			repository.items = append(repository.items[:index], repository.items[index+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (repository *memoryRepository) RecordScan(context.Context, string, ScanSummary) error {
	return nil
}

func TestAddLibrary(t *testing.T) {
	mediaRoot := t.TempDir()
	videoPath := filepath.Join(mediaRoot, "videos")
	if err := os.Mkdir(videoPath, 0o750); err != nil {
		t.Fatalf("create video directory: %v", err)
	}
	repository := &memoryRepository{}
	service := NewService(repository, mediaRoot)

	created, err := service.Add(context.Background(), " Mes vidéos ", videoPath)
	if err != nil {
		t.Fatalf("Add() returned an error: %v", err)
	}
	if created.Name != "Mes vidéos" || created.Path != videoPath {
		t.Fatalf("unexpected library: %#v", created)
	}
	if created.ID == "" {
		t.Fatal("library ID should not be empty")
	}
}

func TestAddLibraryRejectsPathOutsideMediaRoot(t *testing.T) {
	service := NewService(&memoryRepository{}, t.TempDir())

	_, err := service.Add(context.Background(), "Vidéos", t.TempDir())
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected ErrInvalidPath, got %v", err)
	}
}

func TestAddLibraryRejectsDuplicatePath(t *testing.T) {
	mediaRoot := t.TempDir()
	repository := &memoryRepository{}
	service := NewService(repository, mediaRoot)
	if _, err := service.Add(context.Background(), "Vidéos", mediaRoot); err != nil {
		t.Fatalf("first Add() returned an error: %v", err)
	}

	_, err := service.Add(context.Background(), "Doublon", mediaRoot)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

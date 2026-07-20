package scanmanager

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"flex.local/server/internal/library"
	"flex.local/server/internal/media"
)

type stubLibraries struct{}

func (stubLibraries) List(context.Context) ([]library.Library, error) { return nil, nil }

type fixedLibraries struct{ items []library.Library }

func (source fixedLibraries) List(context.Context) ([]library.Library, error) {
	return source.items, nil
}

type countingScanner struct {
	calls atomic.Int32
	err   error
}

func (scanner *countingScanner) Scan(context.Context, string) (media.ScanResult, error) {
	scanner.calls.Add(1)
	return media.ScanResult{Discovered: 2, Indexed: 1, Skipped: 1}, scanner.err
}

func newTestCoordinator(scanner Scanner) *Coordinator {
	coordinator := New("", stubLibraries{}, scanner, slog.New(slog.NewTextHandler(io.Discard, nil)))
	coordinator.ctx, coordinator.cancel = context.WithCancel(context.Background())
	coordinator.debounce = 20 * time.Millisecond
	return coordinator
}

func TestScheduleDebouncesRepeatedEvents(t *testing.T) {
	scanner := &countingScanner{}
	coordinator := newTestCoordinator(scanner)
	defer coordinator.Close()

	coordinator.schedule("library-1")
	coordinator.schedule("library-1")
	coordinator.schedule("library-1")
	if status := coordinator.Status("library-1"); status.State != "pending" {
		t.Fatalf("expected pending status, got %#v", status)
	}

	waitForStatus(t, coordinator, "library-1", "completed")
	if calls := scanner.calls.Load(); calls != 1 {
		t.Fatalf("expected one scan, got %d", calls)
	}
	status := coordinator.Status("library-1")
	if status.Result == nil || status.Result.Indexed != 1 || status.FinishedAt == nil {
		t.Fatalf("unexpected completed status: %#v", status)
	}
}

func TestFailedScanIsExposedInStatus(t *testing.T) {
	scanner := &countingScanner{err: errors.New("probe unavailable")}
	coordinator := newTestCoordinator(scanner)
	defer coordinator.Close()

	coordinator.Trigger("library-1")
	waitForStatus(t, coordinator, "library-1", "failed")
	status := coordinator.Status("library-1")
	if status.LastError != "probe unavailable" || status.Result != nil || status.FinishedAt == nil {
		t.Fatalf("unexpected failed status: %#v", status)
	}
}

func TestWatcherDebouncesSuccessiveFileWrites(t *testing.T) {
	root := t.TempDir()
	libraryPath := filepath.Join(root, "videos")
	if err := os.Mkdir(libraryPath, 0o750); err != nil {
		t.Fatal(err)
	}
	scanner := &countingScanner{}
	coordinator := New(root, fixedLibraries{items: []library.Library{{ID: "library-1", Path: libraryPath}}}, scanner, slog.New(slog.NewTextHandler(io.Discard, nil)))
	coordinator.debounce = 40 * time.Millisecond
	if err := coordinator.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned an error: %v", err)
	}
	defer coordinator.Close()
	waitForCalls(t, scanner, 1)

	videoPath := filepath.Join(libraryPath, "copy.mp4")
	file, err := os.Create(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, chunk := range []string{"first", "second", "third"} {
		if _, err := file.WriteString(chunk); err != nil {
			t.Fatal(err)
		}
		if err := file.Sync(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	waitForCalls(t, scanner, 2)
	time.Sleep(100 * time.Millisecond)
	if calls := scanner.calls.Load(); calls != 2 {
		t.Fatalf("expected one scan for successive writes, got %d total calls", calls)
	}
}

func waitForStatus(t *testing.T, coordinator *Coordinator, libraryID string, expected string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if coordinator.Status(libraryID).State == expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("status did not become %s: %#v", expected, coordinator.Status(libraryID))
}

func waitForCalls(t *testing.T, scanner *countingScanner, expected int32) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if scanner.calls.Load() >= expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("scanner did not reach %d calls; got %d", expected, scanner.calls.Load())
}

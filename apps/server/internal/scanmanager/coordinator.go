package scanmanager

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"flex.local/server/internal/library"
	"flex.local/server/internal/media"
)

const debounceDelay = 2 * time.Second

type Scanner interface {
	Scan(ctx context.Context, libraryID string) (media.ScanResult, error)
}

type LibrarySource interface {
	List(ctx context.Context) ([]library.Library, error)
}

type Status struct {
	State     string
	StartedAt *time.Time
	LastError string
}

type job struct {
	done   chan struct{}
	result media.ScanResult
	err    error
}

type Coordinator struct {
	mediaRoot string
	libraries LibrarySource
	scanner   Scanner
	logger    *slog.Logger

	watcher *fsnotify.Watcher
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	mu       sync.Mutex
	jobs     map[string]*job
	pending  map[string]bool
	statuses map[string]Status
	timers   map[string]*time.Timer
}

func New(mediaRoot string, libraries LibrarySource, scanner Scanner, logger *slog.Logger) *Coordinator {
	return &Coordinator{
		mediaRoot: mediaRoot,
		libraries: libraries,
		scanner:   scanner,
		logger:    logger,
		jobs:      make(map[string]*job),
		pending:   make(map[string]bool),
		statuses:  make(map[string]Status),
		timers:    make(map[string]*time.Timer),
	}
}

func (coordinator *Coordinator) Start(parent context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create media watcher: %w", err)
	}
	coordinator.watcher = watcher
	coordinator.ctx, coordinator.cancel = context.WithCancel(parent)

	if err := coordinator.addDirectoryTree(coordinator.mediaRoot); err != nil {
		_ = watcher.Close()
		coordinator.cancel()
		return fmt.Errorf("watch media root: %w", err)
	}

	coordinator.wg.Add(1)
	go coordinator.watch()

	libraries, err := coordinator.libraries.List(coordinator.ctx)
	if err != nil {
		coordinator.Close()
		return fmt.Errorf("list libraries for startup scan: %w", err)
	}
	for _, item := range libraries {
		coordinator.Trigger(item.ID)
	}
	return nil
}

func (coordinator *Coordinator) Close() {
	if coordinator.cancel == nil {
		return
	}
	coordinator.cancel()
	coordinator.mu.Lock()
	for id, timer := range coordinator.timers {
		timer.Stop()
		delete(coordinator.timers, id)
	}
	coordinator.mu.Unlock()
	if coordinator.watcher != nil {
		_ = coordinator.watcher.Close()
	}
	coordinator.wg.Wait()
}

func (coordinator *Coordinator) Scan(ctx context.Context, libraryID string) (media.ScanResult, error) {
	current := coordinator.startJob(libraryID, false)
	select {
	case <-current.done:
		return current.result, current.err
	case <-ctx.Done():
		return media.ScanResult{}, ctx.Err()
	}
}

func (coordinator *Coordinator) Trigger(libraryID string) {
	coordinator.startJob(libraryID, true)
}

func (coordinator *Coordinator) Status(libraryID string) Status {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	status, exists := coordinator.statuses[libraryID]
	if !exists {
		return Status{State: "idle"}
	}
	return status
}

func (coordinator *Coordinator) startJob(libraryID string, scanAgainIfRunning bool) *job {
	coordinator.mu.Lock()
	if current, exists := coordinator.jobs[libraryID]; exists {
		if scanAgainIfRunning {
			coordinator.pending[libraryID] = true
		}
		coordinator.mu.Unlock()
		return current
	}
	current := &job{done: make(chan struct{})}
	coordinator.jobs[libraryID] = current
	startedAt := time.Now().UTC()
	coordinator.statuses[libraryID] = Status{State: "scanning", StartedAt: &startedAt}
	coordinator.mu.Unlock()

	coordinator.wg.Add(1)
	go func() {
		defer coordinator.wg.Done()
		current.result, current.err = coordinator.scanner.Scan(coordinator.ctx, libraryID)

		status := Status{State: "idle"}
		if current.err != nil {
			status.LastError = current.err.Error()
			coordinator.logger.Error("automatic library scan failed", "library_id", libraryID, "error", current.err)
		} else {
			coordinator.logger.Info("library scan completed", "library_id", libraryID, "discovered", current.result.Discovered, "indexed", current.result.Indexed, "skipped", current.result.Skipped)
		}

		coordinator.mu.Lock()
		delete(coordinator.jobs, libraryID)
		coordinator.statuses[libraryID] = status
		pending := coordinator.pending[libraryID]
		delete(coordinator.pending, libraryID)
		close(current.done)
		coordinator.mu.Unlock()
		if pending && coordinator.ctx.Err() == nil {
			coordinator.Trigger(libraryID)
		}
	}()
	return current
}

func (coordinator *Coordinator) watch() {
	defer coordinator.wg.Done()
	for {
		select {
		case <-coordinator.ctx.Done():
			return
		case event, ok := <-coordinator.watcher.Events:
			if !ok {
				return
			}
			coordinator.handleEvent(event)
		case err, ok := <-coordinator.watcher.Errors:
			if ok {
				coordinator.logger.Error("media watcher error", "error", err)
			}
		}
	}
}

func (coordinator *Coordinator) handleEvent(event fsnotify.Event) {
	if event.Op&fsnotify.Create != 0 {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			if err := coordinator.addDirectoryTree(event.Name); err != nil {
				coordinator.logger.Error("watch new media directory", "path", event.Name, "error", err)
			}
		}
	}
	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
		return
	}

	libraries, err := coordinator.libraries.List(coordinator.ctx)
	if err != nil {
		coordinator.logger.Error("list libraries after media change", "error", err)
		return
	}
	for _, item := range libraries {
		if pathWithin(item.Path, event.Name) {
			coordinator.schedule(item.ID)
		}
	}
}

func (coordinator *Coordinator) schedule(libraryID string) {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if timer, exists := coordinator.timers[libraryID]; exists {
		timer.Reset(debounceDelay)
		return
	}
	coordinator.timers[libraryID] = time.AfterFunc(debounceDelay, func() {
		coordinator.mu.Lock()
		delete(coordinator.timers, libraryID)
		coordinator.mu.Unlock()
		coordinator.Trigger(libraryID)
	})
}

func (coordinator *Coordinator) addDirectoryTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if err := coordinator.watcher.Add(path); err != nil {
				return fmt.Errorf("watch %s: %w", path, err)
			}
		}
		return nil
	})
}

func pathWithin(root string, path string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

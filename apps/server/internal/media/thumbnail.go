package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
)

type ThumbnailGenerator struct {
	cachePath string
	binary    string
	mutex     sync.Mutex
}

func NewThumbnailGenerator(cachePath string, binary string) *ThumbnailGenerator {
	if binary == "" {
		binary = "ffmpeg"
	}
	return &ThumbnailGenerator{cachePath: filepath.Join(cachePath, "thumbnails"), binary: binary}
}

func (generator *ThumbnailGenerator) Generate(ctx context.Context, item File) (string, error) {
	generator.mutex.Lock()
	defer generator.mutex.Unlock()

	if err := os.MkdirAll(generator.cachePath, 0o750); err != nil {
		return "", fmt.Errorf("create thumbnail cache: %w", err)
	}
	target := filepath.Join(generator.cachePath, item.ID+".jpg")
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}

	temporary, err := os.CreateTemp(generator.cachePath, item.ID+"-*.jpg")
	if err != nil {
		return "", fmt.Errorf("create temporary thumbnail: %w", err)
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close temporary thumbnail: %w", err)
	}
	defer os.Remove(temporaryPath)

	seekSeconds := float64(item.DurationMS) / 10_000
	if seekSeconds > 60 {
		seekSeconds = 60
	}
	command := exec.CommandContext(ctx, generator.binary,
		"-v", "error", "-ss", strconv.FormatFloat(seekSeconds, 'f', 3, 64), "-i", item.Path,
		"-frames:v", "1", "-vf", "scale=640:-2", "-q:v", "3", "-y", temporaryPath,
	)
	if output, err := command.CombinedOutput(); err != nil {
		return "", fmt.Errorf("generate thumbnail: %w: %s", err, string(output))
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return "", fmt.Errorf("store thumbnail: %w", err)
	}
	return target, nil
}

func (generator *ThumbnailGenerator) Remove(mediaID string) error {
	generator.mutex.Lock()
	defer generator.mutex.Unlock()
	if mediaID == "" || filepath.Base(mediaID) != mediaID {
		return fmt.Errorf("invalid media id")
	}
	err := os.Remove(filepath.Join(generator.cachePath, mediaID+".jpg"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove thumbnail cache: %w", err)
	}
	return nil
}

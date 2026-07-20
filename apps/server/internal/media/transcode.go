package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type PlaybackMode string

const (
	PlaybackDirect PlaybackMode = "direct"
	PlaybackHLS    PlaybackMode = "hls"
)

func SelectPlaybackMode(item File) PlaybackMode {
	container := strings.ToLower(item.Container)
	codec := strings.ToLower(item.VideoCodec)
	if codec == "h264" && (strings.Contains(container, "mov") || strings.Contains(container, "mp4")) {
		return PlaybackDirect
	}
	if (codec == "vp8" || codec == "vp9") && strings.Contains(container, "webm") {
		return PlaybackDirect
	}
	return PlaybackHLS
}

type Transcoder struct {
	cachePath string
	binary    string
	mutex     sync.Mutex
}

func NewTranscoder(cachePath string, binary string) *Transcoder {
	if binary == "" {
		binary = "ffmpeg"
	}
	return &Transcoder{cachePath: filepath.Join(cachePath, "transcodes"), binary: binary}
}

func (transcoder *Transcoder) Generate(ctx context.Context, item File) (string, error) {
	transcoder.mutex.Lock()
	defer transcoder.mutex.Unlock()

	directory := filepath.Join(transcoder.cachePath, item.ID+"-"+strconv.FormatInt(item.ModifiedAt.Unix(), 10))
	playlist := filepath.Join(directory, "index.m3u8")
	if _, err := os.Stat(playlist); err == nil {
		return directory, nil
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return "", fmt.Errorf("create transcode cache: %w", err)
	}

	temporaryDirectory, err := os.MkdirTemp(transcoder.cachePath, item.ID+"-*")
	if err != nil {
		return "", fmt.Errorf("create temporary transcode directory: %w", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	temporaryPlaylist := filepath.Join(temporaryDirectory, "index.m3u8")
	segmentPattern := filepath.Join(temporaryDirectory, "segment-%05d.ts")
	command := exec.CommandContext(ctx, transcoder.binary,
		"-v", "error", "-i", item.Path,
		"-map", "0:v:0", "-map", "0:a:0?",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "22", "-pix_fmt", "yuv420p",
		"-vf", "scale='min(1920,iw)':-2",
		"-c:a", "aac", "-b:a", "192k",
		"-force_key_frames", "expr:gte(t,n_forced*4)",
		"-f", "hls", "-hls_time", "4", "-hls_playlist_type", "vod",
		"-hls_segment_filename", segmentPattern, temporaryPlaylist,
	)
	if output, err := command.CombinedOutput(); err != nil {
		return "", fmt.Errorf("transcode media: %w: %s", err, string(output))
	}
	if err := os.RemoveAll(directory); err != nil {
		return "", fmt.Errorf("replace transcode cache: %w", err)
	}
	if err := os.Rename(temporaryDirectory, directory); err != nil {
		return "", fmt.Errorf("store transcode: %w", err)
	}
	return directory, nil
}

func (transcoder *Transcoder) Remove(mediaID string) error {
	transcoder.mutex.Lock()
	defer transcoder.mutex.Unlock()
	if mediaID == "" || filepath.Base(mediaID) != mediaID {
		return fmt.Errorf("invalid media id")
	}
	entries, err := os.ReadDir(transcoder.cachePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read transcode cache: %w", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), mediaID+"-") {
			if err := os.RemoveAll(filepath.Join(transcoder.cachePath, entry.Name())); err != nil {
				return fmt.Errorf("remove transcode cache: %w", err)
			}
		}
	}
	return nil
}

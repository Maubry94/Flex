package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type FFprobe struct{ Binary string }

type probeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"streams"`
	Format struct {
		Duration   string `json:"duration"`
		FormatName string `json:"format_name"`
	} `json:"format"`
}

func (probe FFprobe) Analyze(ctx context.Context, path string) (TechnicalMetadata, error) {
	binary := probe.Binary
	if binary == "" {
		binary = "ffprobe"
	}
	output, err := exec.CommandContext(ctx, binary, "-v", "error", "-show_streams", "-show_format", "-of", "json", path).Output()
	if err != nil {
		return TechnicalMetadata{}, fmt.Errorf("run ffprobe: %w", err)
	}
	var parsed probeOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return TechnicalMetadata{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	duration, _ := strconv.ParseFloat(parsed.Format.Duration, 64)
	metadata := TechnicalMetadata{DurationMS: int64(duration * 1000), Container: parsed.Format.FormatName}
	for _, stream := range parsed.Streams {
		switch stream.CodecType {
		case "video":
			if metadata.VideoCodec == "" {
				metadata.VideoCodec, metadata.Width, metadata.Height = stream.CodecName, stream.Width, stream.Height
			}
		case "audio":
			if metadata.AudioCodec == "" {
				metadata.AudioCodec = stream.CodecName
			}
		}
	}
	if metadata.VideoCodec == "" {
		return TechnicalMetadata{}, fmt.Errorf("no video stream found")
	}
	return metadata, nil
}

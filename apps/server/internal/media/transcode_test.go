package media

import "testing"

func TestSelectPlaybackMode(t *testing.T) {
	tests := []struct {
		name     string
		file     File
		expected PlaybackMode
	}{
		{name: "H264 MP4 is direct", file: File{VideoCodec: "h264", Container: "mov,mp4,m4a"}, expected: PlaybackDirect},
		{name: "HEVC MOV uses HLS", file: File{VideoCodec: "hevc", Container: "mov,mp4,m4a"}, expected: PlaybackHLS},
		{name: "H264 MKV uses HLS", file: File{VideoCodec: "h264", Container: "matroska,webm"}, expected: PlaybackHLS},
		{name: "VP9 WebM is direct", file: File{VideoCodec: "vp9", Container: "matroska,webm"}, expected: PlaybackDirect},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := SelectPlaybackMode(test.file); actual != test.expected {
				t.Fatalf("SelectPlaybackMode() = %s, want %s", actual, test.expected)
			}
		})
	}
}

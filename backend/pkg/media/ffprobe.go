package media

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type VideoMeta struct {
	Duration   float64
	Width      int
	Height     int
	IsVertical bool
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Duration  string `json:"duration"`
}

func ExtractVideoMeta(filepath string) (*VideoMeta, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filepath,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result ffprobeOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	var videoStream *ffprobeStream
	var duration float64

	for i := range result.Streams {
		s := &result.Streams[i]
		if s.CodecType == "video" && videoStream == nil {
			videoStream = s
			if s.Duration != "" {
				fmt.Sscanf(s.Duration, "%f", &duration)
			}
		}
		if s.CodecType == "audio" && duration == 0 && s.Duration != "" {
			fmt.Sscanf(s.Duration, "%f", &duration)
		}
	}

	if videoStream == nil {
		return nil, fmt.Errorf("no video stream found")
	}

	if duration == 0 {
		duration = parseFormatDuration(out)
	}

	return &VideoMeta{
		Duration:   duration,
		Width:      videoStream.Width,
		Height:     videoStream.Height,
		IsVertical: videoStream.Width < videoStream.Height,
	}, nil
}

func parseFormatDuration(output []byte) float64 {
	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return 0
	}
	if result.Format.Duration == "" {
		return 0
	}
	var d float64
	if _, err := fmt.Sscanf(result.Format.Duration, "%f", &d); err != nil {
		return 0
	}
	return d
}

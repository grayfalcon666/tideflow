package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Video struct {
	ID        uint           `gorm:"primaryKey"`
	PlayURL   string         `gorm:"size:255"`
	Duration  float64       `gorm:"type:float;default:0"`
	Width     int           `gorm:"type:int;default:0"`
	Height    int           `gorm:"type:int;default:0"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Video) TableName() string { return "videos" }

type VideoMeta struct {
	Duration   float64
	Width      int
	Height     int
	IsVertical bool
}

type ffprobeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Duration  string `json:"duration"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func extractMeta(fp string) (*VideoMeta, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		fp,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result ffprobeOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	var videoIdx int = -1
	var duration float64

	for i := range result.Streams {
		s := &result.Streams[i]
		if s.CodecType == "video" && videoIdx == -1 {
			videoIdx = i
			if s.Duration != "" {
				fmt.Sscanf(s.Duration, "%f", &duration)
			}
		}
		if s.CodecType == "audio" && duration == 0 && s.Duration != "" {
			fmt.Sscanf(s.Duration, "%f", &duration)
		}
	}

	if videoIdx == -1 {
		return nil, fmt.Errorf("no video stream found")
	}

	if duration == 0 && result.Format.Duration != "" {
		fmt.Sscanf(result.Format.Duration, "%f", &duration)
	}

	return &VideoMeta{
		Duration:   duration,
		Width:      result.Streams[videoIdx].Width,
		Height:     result.Streams[videoIdx].Height,
		IsVertical: result.Streams[videoIdx].Width < result.Streams[videoIdx].Height,
	}, nil
}

func main() {
	dsn := "root:password@tcp(localhost:3306)/tideflow?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	var videos []Video
	if err := db.Where("duration = 0 OR duration IS NULL").Find(&videos).Error; err != nil {
		log.Fatalf("query videos: %v", err)
	}

	if len(videos) == 0 {
		fmt.Println("No videos need backfill")
		return
	}

	fmt.Printf("Found %d videos to backfill\n", len(videos))

	uploadDir := "./uploads"
	for _, v := range videos {
		if v.PlayURL == "" {
			continue
		}

		filename := filepath.Base(v.PlayURL)
		fp := filepath.Join(uploadDir, "videos", filename)

		if _, err := os.Stat(fp); os.IsNotExist(err) {
			fmt.Printf("File not found: %s\n", fp)
			continue
		}

		fmt.Printf("Processing video %d: %s\n", v.ID, filename)

		meta, err := extractMeta(fp)
		if err != nil {
			fmt.Printf("  ffprobe failed: %v\n", err)
			continue
		}

		fmt.Printf("  duration=%.2f, width=%d, height=%d, is_vertical=%v\n",
			meta.Duration, meta.Width, meta.Height, meta.IsVertical)

		if err := db.Model(&Video{}).Where("id = ?", v.ID).Updates(map[string]interface{}{
			"duration": meta.Duration,
			"width":    meta.Width,
			"height":   meta.Height,
		}).Error; err != nil {
			fmt.Printf("  update failed: %v\n", err)
			continue
		}

		fmt.Printf("  Updated successfully\n")
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("Done")
}

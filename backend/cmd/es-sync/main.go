package main

import (
	"context"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	infraes "tideflow/infra/es"
	"tideflow/internal/config"
	"tideflow/internal/models"
)

func main() {
	cfg, err := config.Load("../.env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Elasticsearch.Addresses) == 0 {
		log.Fatal("ES_ADDRESSES not configured")
	}

	db, err := gorm.Open(mysql.Open(cfg.DB.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	esClient, err := infraes.NewClient(&cfg.Elasticsearch)
	if err != nil {
		log.Fatalf("failed to connect ES: %v", err)
	}

	ctx := context.Background()

	if err := esClient.EnsureIndex(ctx); err != nil {
		log.Fatalf("failed to ensure ES index: %v", err)
	}

	// Load all non-deleted videos
	var videos []models.Video
	if err := db.Find(&videos).Error; err != nil {
		log.Fatalf("failed to load videos: %v", err)
	}
	log.Printf("loaded %d videos from database", len(videos))

	// Load all tags into map: videoID -> []tagName
	type videoTagRow struct {
		VideoID uint
		Name    string
	}
	var tagRows []videoTagRow
	db.Raw(`SELECT vt.video_id, t.name FROM video_tags vt
		JOIN tags t ON vt.tag_id = t.id`).Scan(&tagRows)

	tagsMap := make(map[uint][]string)
	for _, r := range tagRows {
		tagsMap[r.VideoID] = append(tagsMap[r.VideoID], r.Name)
	}

	// Build ES docs
	docs := make([]*infraes.VideoDoc, 0, len(videos))
	for _, v := range videos {
		docs = append(docs, &infraes.VideoDoc{
			VideoID:     v.ID,
			Title:       v.Title,
			Description: v.Description,
			Username:    v.Username,
			Tags:        tagsMap[v.ID],
			Popularity:  v.Popularity,
			CreateTime:  v.CreateTime.UnixMilli(),
		})
	}

	// Bulk upsert in batches of 500
	batchSize := 500
	total := len(docs)
	synced := 0
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		batch := docs[i:end]
		if err := esClient.BulkUpsert(ctx, batch); err != nil {
			log.Printf("batch %d-%d failed: %v", i, end, err)
			continue
		}
		synced += len(batch)
		fmt.Printf("\rsynced %d/%d", synced, total)
	}

	// Refresh index so documents are searchable immediately
	esClient.RefreshIndex(ctx)

	fmt.Printf("\n\ndone! synced %d/%d videos to ES index\n", synced, total)
}

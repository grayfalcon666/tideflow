package main

import (
	"log"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tideflow/internal/config"
	"tideflow/internal/models"
)

func main() {
	cfg, err := config.Load("../.env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := gorm.Open(mysql.Open(cfg.DB.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := migrate(db); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	log.Println("migration completed successfully")
}

func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Account{},
		&models.Video{},
		&models.Like{},
		&models.Comment{},
		&models.Social{},
		&models.Message{},
		&models.Tag{},
		&models.VideoTag{},
		&models.OutboxMsg{},
		&models.Notification{},
		&models.Note{},
	); err != nil {
		return err
	}

	// 复合索引：加速 Timeline 回源和热门排序查询
	indexes := []string{
		"CREATE INDEX idx_videos_author_create ON videos (author_id, create_time)",
		"CREATE INDEX idx_videos_likes_count_id ON videos (likes_count, id)",
		"CREATE INDEX idx_videos_popularity_time_id ON videos (popularity, create_time)",
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			// 忽略 "duplicate index name" 错误
			if !isDuplicateIndexErr(err) {
				return err
			}
		}
	}
	return nil
}

func isDuplicateIndexErr(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate key name") || strings.Contains(err.Error(), "index already exists"))
}
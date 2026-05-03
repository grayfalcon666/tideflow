package main

import (
	"log"

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
	return db.AutoMigrate(
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
	)
}
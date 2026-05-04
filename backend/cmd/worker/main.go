package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tideflow/internal/config"
	"tideflow/internal/mq"
	"tideflow/internal/repository"
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

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Host,
		Password: cfg.Redis.Password,
		DB:   cfg.Redis.DB,
	})

	mqInstance, err := mq.NewMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("failed to connect to MQ: %v", err)
	}
	defer mqInstance.Close()

	repo := repository.New(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		tw := mq.NewTimelineWorker(mqInstance, rdb, repo, cfg.BigVThreshold)
		tw.Start(ctx)
	}()

	go func() {
		lw := mq.NewLikeWorker(mqInstance, repo, rdb, cfg.BigVThreshold)
		lw.Start(ctx)
	}()

	go func() {
		cw := mq.NewCommentWorker(mqInstance, rdb, repo)
		cw.Start(ctx)
	}()

	go func() {
		sw := mq.NewSocialWorker(mqInstance, rdb, repo, cfg.BigVThreshold)
		sw.Start(ctx)
	}()

	go func() {
		pw := mq.NewPopularityWorker(mqInstance, rdb)
		pw.Start(ctx)
	}()

	go func() {
		ow := mq.NewOutboxWorker(mqInstance, repo)
		ow.Start(ctx)
	}()

	hub := mq.NewSSEHub()
	go hub.Run()

	go func() {
		nw := mq.NewNotificationWorker(mqInstance, hub, repo)
		nw.Start(ctx)
	}()

	log.Println("worker started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("worker shutting down...")
	cancel()
	log.Println("worker stopped")
}
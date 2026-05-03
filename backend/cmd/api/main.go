package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tideflow/internal/config"
	_ "tideflow/internal/docs"
	"tideflow/internal/handler"
	"tideflow/internal/middleware"
	"tideflow/internal/mq"
	"tideflow/internal/repository"
	"tideflow/internal/service"
	infraredis "tideflow/infra/redis"
)

func main() {
	cfg, err := config.Load("../.env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := initDB(cfg)
	if err != nil {
		log.Fatalf("failed to init db: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Host,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	mqInstance, err := mq.NewMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Printf("warning: failed to connect to MQ: %v", err)
	}

	cache := infraredis.NewCache(rdb)
	rateLimiter := infraredis.NewRateLimiter(rdb)

	repo := repository.New(db)
	authSvc := service.NewAuthService(repo, cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)
	userSvc := service.NewUserService(repo, cfg.BigVThreshold)
	videoSvc := service.NewVideoService(repo, cache, cfg.BigVThreshold, cfg.Upload.Dir)
	feedSvc := service.NewFeedService(repo, cache, rdb, cfg.BigVThreshold)
	interactionSvc := service.NewInteractionService(repo)
	msgSvc := service.NewMessageService(repo)
	notifSvc := service.NewNotificationService(repo)

	authMw := middleware.NewAuthMiddleware(authSvc)
	rateLimitMw := middleware.NewRateLimitMiddleware(rateLimiter)

	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userSvc)
	videoHandler := handler.NewVideoHandler(videoSvc)
	feedHandler := handler.NewFeedHandler(feedSvc)
	interactionHandler := handler.NewInteractionHandler(interactionSvc)
	msgHandler := handler.NewMessageHandler(msgSvc)
	notifHandler := handler.NewNotificationHandler(notifSvc)
	sseHandler := handler.NewSSEHandler(nil)
	tagHandler := handler.NewTagHandler()

	router := setupRouter(authMw, rateLimitMw, authHandler, userHandler, videoHandler, feedHandler, interactionHandler, msgHandler, notifHandler, sseHandler, tagHandler)

	hub := mq.NewSSEHub()
	go hub.Run()
	sseHandler = handler.NewSSEHandler(hub)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	go func() {
		log.Printf("server starting on :%s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	if mqInstance != nil {
		mqInstance.Close()
	}

	log.Println("server exited")
}

func initDB(cfg *config.Config) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(cfg.DB.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
}

func setupRouter(
	authMw *middleware.AuthMiddleware,
	rateLimitMw *middleware.RateLimitMiddleware,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	videoHandler *handler.VideoHandler,
	feedHandler *handler.FeedHandler,
	interactionHandler *handler.InteractionHandler,
	msgHandler *handler.MessageHandler,
	notifHandler *handler.NotificationHandler,
	sseHandler *handler.SSEHandler,
	tagHandler *handler.TagHandler,
) *gin.Engine {
	r := gin.Default()

	r.POST("/api/v1/auth/register", rateLimitMw.RegisterLimit(), authHandler.Register)
	r.POST("/api/v1/auth/login", rateLimitMw.LoginLimit(), authHandler.Login)
	r.POST("/api/v1/auth/refresh", authHandler.Refresh)
	r.POST("/api/v1/auth/password", authHandler.ChangePassword)
	r.POST("/api/v1/auth/logout", authMw.JWTAuth(), authHandler.Logout)

	r.GET("/api/v1/users/me", authMw.JWTAuth(), userHandler.GetMe)
	r.GET("/api/v1/users/:id", userHandler.GetUser)
	r.GET("/api/v1/users/username/:username", userHandler.GetUserByUsername)
	r.PUT("/api/v1/users/me", authMw.JWTAuth(), userHandler.UpdateMe)
	r.PUT("/api/v1/users/me/username", authMw.JWTAuth(), userHandler.UpdateUsername)
	r.GET("/api/v1/users/:id/videos", authMw.SoftJWTAuth(), userHandler.GetUserVideos)

	r.POST("/api/v1/users/:user_id/follow", authMw.JWTAuth(), userHandler.Follow)
	r.DELETE("/api/v1/users/:user_id/follow", authMw.JWTAuth(), userHandler.Unfollow)
	r.GET("/api/v1/users/:id/following", authMw.JWTAuth(), userHandler.GetFollowing)
	r.GET("/api/v1/users/:id/followers", authMw.JWTAuth(), userHandler.GetFollowers)
	r.GET("/api/v1/users/:id/social-counts", authMw.JWTAuth(), userHandler.GetSocialCounts)

	r.POST("/api/v1/videos/upload", authMw.JWTAuth(), videoHandler.UploadVideo)
	r.POST("/api/v1/videos/cover", authMw.JWTAuth(), videoHandler.UploadCover)
	r.POST("/api/v1/videos", authMw.JWTAuth(), videoHandler.PublishVideo)
	r.GET("/api/v1/videos/:id", authMw.SoftJWTAuth(), videoHandler.GetVideo)
	r.PUT("/api/v1/videos/:id", authMw.JWTAuth(), videoHandler.UpdateVideo)
	r.DELETE("/api/v1/videos/:id", authMw.JWTAuth(), videoHandler.DeleteVideo)

	r.GET("/api/v1/feed/latest", authMw.SoftJWTAuth(), feedHandler.ListLatest)
	r.GET("/api/v1/feed/popular", authMw.SoftJWTAuth(), feedHandler.ListPopular)
	r.GET("/api/v1/feed/following", authMw.JWTAuth(), feedHandler.ListByFollowing)
	r.GET("/api/v1/feed/tag", authMw.SoftJWTAuth(), feedHandler.ListByTag)

	r.POST("/api/v1/videos/:id/like", authMw.JWTAuth(), interactionHandler.LikeVideo)
	r.DELETE("/api/v1/videos/:id/like", authMw.JWTAuth(), interactionHandler.UnlikeVideo)
	r.GET("/api/v1/videos/:id/like", authMw.JWTAuth(), interactionHandler.IsLiked)
	r.GET("/api/v1/likes/mine", authMw.JWTAuth(), interactionHandler.GetLikedVideos)
	r.POST("/api/v1/videos/:id/comments", authMw.JWTAuth(), interactionHandler.PublishComment)
	r.DELETE("/api/v1/videos/:id/comments/:comment_id", authMw.JWTAuth(), interactionHandler.DeleteComment)
	r.GET("/api/v1/videos/:id/comments", authMw.SoftJWTAuth(), interactionHandler.GetComments)

	r.POST("/api/v1/messages", authMw.JWTAuth(), msgHandler.SendMessage)
	r.GET("/api/v1/messages/conversations", authMw.JWTAuth(), msgHandler.GetConversations)
	r.GET("/api/v1/messages/conversations/:user_id", authMw.JWTAuth(), msgHandler.GetMessages)
	r.PUT("/api/v1/messages/conversations/:user_id/read", authMw.JWTAuth(), msgHandler.MarkRead)

	r.GET("/api/v1/notifications", authMw.JWTAuth(), notifHandler.GetNotifications)
	r.PUT("/api/v1/notifications/read", authMw.JWTAuth(), notifHandler.MarkRead)
	r.GET("/api/v1/notifications/unread-count", authMw.JWTAuth(), notifHandler.UnreadCount)
	r.GET("/api/v1/notifications/stream", authMw.SSERequireAuth(), sseHandler.Stream)

	r.GET("/api/v1/tags/hot", tagHandler.GetHotTags)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
// TideFlow API
//
// @title TideFlow 短视频 Feed 流 API
// @version v1
// @description 高性能短视频 Feed 流系统 API 文档
// @termsOfService http://swagger.io/terms/
//
// @securityDefinitions.oauth2.password OAuth2Password
// @tokenUrl /api/v1/auth/oauth2/token
// @scope.read Grant read access
// @scope.write Grant write access
//
// @host localhost:8080
// @BasePath /
// @schemes http https
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	infraes "tideflow/infra/es"
	infraredis "tideflow/infra/redis"
	"tideflow/internal/config"
	_ "tideflow/internal/docs"
	"tideflow/internal/handler"
	"tideflow/internal/middleware"
	"tideflow/internal/mq"
	"tideflow/internal/repository"
	"tideflow/internal/service"
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
		log.Fatalf("failed to connect to MQ: %v", err)
	}

	cache := infraredis.NewCache(rdb)
	rateLimiter := infraredis.NewRateLimiter(rdb)

	repo := repository.New(db)
	cache.SetRepo(repo)
	authSvc := service.NewAuthService(repo, cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)
	userSvc := service.NewUserService(repo, cfg.BigVThreshold, mqInstance)
	videoSvc := service.NewVideoService(repo, cache, cfg.BigVThreshold, cfg.Upload.Dir, cfg.JWT.Secret, mqInstance)
	feedSvc := service.NewFeedService(repo, cache, rdb, cfg.BigVThreshold)
	interactionSvc := service.NewInteractionService(repo, mqInstance)
	msgSvc := service.NewMessageService(repo)
	notifSvc := service.NewNotificationService(repo)
	tagSvc := service.NewTagService(repo)
	noteSvc := service.NewNoteService(repo)
	wordbankSvc := service.NewWordbankService(repo, cache)

	// Scan vocab dir on startup: import new files, skip existing, delete removed
	historySvc := service.NewHistoryService(repo, rdb)
	learningSvc := service.NewLearningService(repo, wordbankSvc, cache)
	if err := learningSvc.ImportVocabLists(context.Background(), cfg.VocabListsDir); err != nil {
		log.Printf("warning: vocab import failed: %v", err)
	}

	var searchSvc *service.SearchService
	var searchHandler *handler.SearchHandler
	if len(cfg.Elasticsearch.Addresses) > 0 {
		esClient, err := infraes.NewClient(&cfg.Elasticsearch)
		if err != nil {
			log.Printf("failed to connect to ES, search disabled: %v", err)
		} else {
			if err := esClient.EnsureIndex(context.Background()); err != nil {
				log.Printf("failed to ensure ES index: %v", err)
			} else {
				log.Println("ES connected and index ready")
				searchSvc = service.NewSearchService(esClient, cache, repo, cfg.BigVThreshold)
				searchHandler = handler.NewSearchHandler(searchSvc)
			}
		}
	}

	authMw := middleware.NewAuthMiddleware(authSvc)
	rateLimitMw := middleware.NewRateLimitMiddleware(rateLimiter)

	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userSvc)
	videoHandler := handler.NewVideoHandler(videoSvc)
	feedHandler := handler.NewFeedHandler(feedSvc)
	interactionHandler := handler.NewInteractionHandler(interactionSvc, userSvc)
	msgHandler := handler.NewMessageHandler(msgSvc)
	notifHandler := handler.NewNotificationHandler(notifSvc)
	tagHandler := handler.NewTagHandler(tagSvc)
	noteHandler := handler.NewNoteHandler(noteSvc, userSvc)
	wordbankHandler := handler.NewWordbankHandler(wordbankSvc)
	learningHandler := handler.NewLearningHandler(learningSvc)
	adminVocabHandler := handler.NewAdminVocabHandler(learningSvc, cfg.VocabListsDir)
	historyHandler := handler.NewHistoryHandler(historySvc)

	hub := mq.NewSSEHub()
	go hub.Run()
	sseHandler := handler.NewSSEHandler(hub)

	router := setupRouter(authMw, rateLimitMw, authHandler, userHandler, videoHandler, feedHandler, interactionHandler, msgHandler, notifHandler, sseHandler, tagHandler, noteHandler, wordbankHandler, searchHandler, learningHandler, adminVocabHandler, historyHandler, cfg)

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
	noteHandler *handler.NoteHandler,
	wordbankHandler *handler.WordbankHandler,
	searchHandler *handler.SearchHandler,
	learningHandler *handler.LearningHandler,
	adminVocabHandler *handler.AdminVocabHandler,
	historyHandler *handler.HistoryHandler,
	cfg *config.Config,
) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 静态文件：视频和封面
	r.Static("/videos", cfg.Upload.Dir+"/videos")
	r.Static("/covers", cfg.Upload.Dir+"/covers")

	r.POST("/api/v1/auth/register", rateLimitMw.RegisterLimit(), authHandler.Register)
	r.POST("/api/v1/auth/login", rateLimitMw.LoginLimit(), authHandler.Login)
	r.POST("/api/v1/auth/oauth2/token", authHandler.OAuth2Token)
	r.POST("/api/v1/auth/refresh", authHandler.Refresh)
	r.POST("/api/v1/auth/password", authHandler.ChangePassword)
	r.POST("/api/v1/auth/logout", authMw.JWTAuth(), authHandler.Logout)

	r.GET("/api/v1/users/me", authMw.JWTAuth(), userHandler.GetMe)
	r.GET("/api/v1/users/:id", authMw.SoftJWTAuth(), userHandler.GetUser)
	r.GET("/api/v1/users/username/:username", userHandler.GetUserByUsername)
	r.PUT("/api/v1/users/me", authMw.JWTAuth(), userHandler.UpdateMe)
	r.PUT("/api/v1/users/me/username", authMw.JWTAuth(), userHandler.UpdateUsername)
	r.GET("/api/v1/users/:id/videos", authMw.SoftJWTAuth(), userHandler.GetUserVideos)

	r.POST("/api/v1/users/:user_id/follow", authMw.JWTAuth(), rateLimitMw.SocialLimit(), userHandler.Follow)
	r.DELETE("/api/v1/users/:user_id/follow", authMw.JWTAuth(), rateLimitMw.SocialLimit(), userHandler.Unfollow)
	r.GET("/api/v1/users/:id/following", authMw.JWTAuth(), userHandler.GetFollowing)
	r.GET("/api/v1/users/:id/followers", authMw.JWTAuth(), userHandler.GetFollowers)
	r.GET("/api/v1/users/:id/social-counts", authMw.JWTAuth(), userHandler.GetSocialCounts)

	r.POST("/api/v1/videos/upload", authMw.JWTAuth(), videoHandler.UploadVideo)
	r.POST("/api/v1/videos/cover", authMw.JWTAuth(), videoHandler.UploadCover)
	r.POST("/api/v1/videos", authMw.JWTAuth(), videoHandler.PublishVideo)
	r.GET("/api/v1/videos/:id", authMw.JWTAuth(), videoHandler.GetVideo)
	r.PUT("/api/v1/videos/:id", authMw.JWTAuth(), videoHandler.UpdateVideo)
	r.DELETE("/api/v1/videos/:id", authMw.JWTAuth(), videoHandler.DeleteVideo)

	// 切片上传（分块 / 断点续传 / 并发）
	r.POST("/api/v1/videos/upload/init", authMw.JWTAuth(), videoHandler.InitChunkedUpload)
	r.POST("/api/v1/videos/upload/chunk", authMw.JWTAuth(), videoHandler.UploadChunk)
	r.GET("/api/v1/videos/upload/status/:upload_id", authMw.JWTAuth(), videoHandler.GetUploadStatus)
	r.POST("/api/v1/videos/upload/complete", authMw.JWTAuth(), videoHandler.CompleteChunkedUpload)

	r.GET("/api/v1/feed/latest", authMw.SoftJWTAuth(), feedHandler.ListLatest)
	r.GET("/api/v1/feed/popular", authMw.SoftJWTAuth(), feedHandler.ListPopular)
	r.GET("/api/v1/feed/following", authMw.JWTAuth(), feedHandler.ListByFollowing)
	r.GET("/api/v1/feed/tag", authMw.SoftJWTAuth(), feedHandler.ListByTag)

	r.POST("/api/v1/videos/:id/like", authMw.JWTAuth(), rateLimitMw.LikeLimit(), interactionHandler.LikeVideo)
	r.DELETE("/api/v1/videos/:id/like", authMw.JWTAuth(), rateLimitMw.LikeLimit(), interactionHandler.UnlikeVideo)
	r.GET("/api/v1/videos/:id/like", authMw.JWTAuth(), interactionHandler.IsLiked)
	r.GET("/api/v1/likes/mine", authMw.JWTAuth(), interactionHandler.GetLikedVideos)
	r.GET("/api/v1/users/:id/liked-videos", authMw.SoftJWTAuth(), interactionHandler.GetUserLikedVideos)
	r.POST("/api/v1/videos/:id/comments", authMw.JWTAuth(), rateLimitMw.CommentLimit(), interactionHandler.PublishComment)
	r.DELETE("/api/v1/videos/:id/comments/:comment_id", authMw.JWTAuth(), rateLimitMw.CommentLimit(), interactionHandler.DeleteComment)
	r.GET("/api/v1/videos/:id/comments", authMw.SoftJWTAuth(), interactionHandler.GetComments)

	r.POST("/api/v1/videos/:id/notes", authMw.JWTAuth(), rateLimitMw.NoteLimit(), noteHandler.CreateNote)
	r.GET("/api/v1/videos/:id/notes", authMw.SoftJWTAuth(), noteHandler.GetNotes)
	r.DELETE("/api/v1/videos/:id/notes/:note_id", authMw.JWTAuth(), noteHandler.DeleteNote)

	r.GET("/api/v1/videos/:id/wordbank", authMw.JWTAuth(), wordbankHandler.GetWordbank)
	r.GET("/api/v1/videos/:id/wordbank/export", authMw.JWTAuth(), wordbankHandler.ExportWordbank)

	// 学习接口
	r.GET("/api/v1/learn/lists", authMw.JWTAuth(), learningHandler.GetLists)
	r.GET("/api/v1/videos/:id/learn/words", authMw.JWTAuth(), learningHandler.GetLearningWords)
	r.GET("/api/v1/videos/:id/learn/word/:word/captions", authMw.JWTAuth(), learningHandler.GetWordCaptions)

	// 学习追踪
	r.POST("/api/v1/learn/commit", authMw.JWTAuth(), learningHandler.CommitLearning)
	r.POST("/api/v1/learn/batch/abort", authMw.JWTAuth(), learningHandler.AbortBatch)
	r.GET("/api/v1/learn/habit/stats", authMw.JWTAuth(), learningHandler.GetHabitStats)
	r.GET("/api/v1/learn/today/words", authMw.JWTAuth(), learningHandler.GetTodayWords)

	// 管理接口
	r.POST("/api/v1/admin/vocab/import", authMw.JWTAuth(), adminVocabHandler.ImportVocabLists)
	r.POST("/api/v1/admin/vocab/reload", authMw.JWTAuth(), adminVocabHandler.ReloadVocab)

	r.POST("/api/v1/messages", authMw.JWTAuth(), msgHandler.SendMessage)
	r.GET("/api/v1/messages/conversations", authMw.JWTAuth(), msgHandler.GetConversations)
	r.GET("/api/v1/messages/conversations/:user_id", authMw.JWTAuth(), msgHandler.GetMessages)
	r.PUT("/api/v1/messages/conversations/:user_id/read", authMw.JWTAuth(), msgHandler.MarkRead)
	r.POST("/api/v1/messages/share-video", authMw.JWTAuth(), msgHandler.ShareVideo)

	r.GET("/api/v1/notifications", authMw.JWTAuth(), notifHandler.GetNotifications)
	r.PUT("/api/v1/notifications/read", authMw.JWTAuth(), notifHandler.MarkRead)
	r.GET("/api/v1/notifications/unread-count", authMw.JWTAuth(), notifHandler.UnreadCount)
	r.GET("/api/v1/notifications/stream", authMw.SSERequireAuth(), sseHandler.Stream)

	r.GET("/api/v1/tags/hot", tagHandler.GetHotTags)

	if searchHandler != nil {
		r.GET("/api/v1/search/videos", authMw.JWTAuth(), searchHandler.SearchVideos)
	}

	r.POST("/api/v1/metrics/view", videoHandler.RecordView)

	r.POST("/api/v1/history/record", authMw.JWTAuth(), historyHandler.RecordHistory)
	r.GET("/api/v1/history", authMw.JWTAuth(), historyHandler.GetHistory)
	r.DELETE("/api/v1/history/:video_id", authMw.JWTAuth(), historyHandler.DeleteHistory)
	r.DELETE("/api/v1/history", authMw.JWTAuth(), historyHandler.ClearHistory)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.PersistAuthorization(true),
	))

	return r
}

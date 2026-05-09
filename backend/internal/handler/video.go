package handler

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
	"tideflow/internal/service"
	"tideflow/pkg/media"
	"tideflow/pkg/response"
)

type VideoHandler struct {
	video *service.VideoService
}

func NewVideoHandler(video *service.VideoService) *VideoHandler {
	return &VideoHandler{video: video}
}

// @Summary 上传视频文件
// @Description 上传视频文件，返回播放URL
// @Tags 视频
// @Security OAuth2Password
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "视频文件"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/videos/upload [post]
func (h *VideoHandler) UploadVideo(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "missing video file")
		return
	}
	if file.Size > 500*1024*1024 {
		response.BadRequest(c, "file too large, max 500MB")
		return
	}

	url, meta, err := h.video.UploadVideo(c.Request.Context(), file)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Created(c, gin.H{
		"play_url":     url,
		"duration":     meta.Duration,
		"width":        meta.Width,
		"height":       meta.Height,
		"is_vertical":  meta.IsVertical,
	})
}

// @Summary 上传封面
// @Description 上传视频封面图片，返回封面URL
// @Tags 视频
// @Security OAuth2Password
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "封面图片"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/videos/cover [post]
func (h *VideoHandler) UploadCover(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "missing cover file")
		return
	}

	url, err := h.video.UploadCover(c.Request.Context(), file)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Created(c, gin.H{"cover_url": url})
}

// @Summary 发布视频
// @Description 发布一个新视频
// @Tags 视频
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param body body PublishVideoRequest true "视频信息"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/videos [post]
func (h *VideoHandler) PublishVideo(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Title       string   `json:"title" binding:"required,min=1,max=255"`
		Description string   `json:"description" binding:"max=500"`
		PlayURL     string   `json:"play_url" binding:"required,min=1"`
		CoverURL    string   `json:"cover_url" binding:"required,min=1"`
		Tags        []string `json:"tags" binding:"max=10,dive,min=1,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	username := ""
	account, err := h.video.GetAccountByID(c.Request.Context(), userID)
	if err == nil && account != nil {
		username = account.Username
	}

	// Reconstruct file path from play_url to extract meta via ffprobe
	// play_url like "/videos/xxx.mp4" -> "./uploads/videos/xxx.mp4"
	filename := req.PlayURL
	if filename != "" && filename[0] == '/' {
		filename = filepath.Join(h.video.UploadDir, "videos", filepath.Base(filename))
	}
	meta, err := media.ExtractVideoMeta(filename)
	if err != nil || meta == nil {
		meta = &media.VideoMeta{}
	}

	videoID, err := h.video.PublishVideo(c.Request.Context(), userID, username, req.Title, req.Description, req.PlayURL, req.CoverURL, req.Tags, meta)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Created(c, gin.H{"video_id": videoID})
}

// @Summary 获取视频详情
// @Description 通过视频ID获取视频详细信息
// @Tags 视频
// @Produce json
// @Param id path int true "视频ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/videos/{id} [get]
func (h *VideoHandler) GetVideo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	video, err := h.video.GetVideoByID(c.Request.Context(), uint(id))
	if err != nil {
		// 尝试 Unscoped 查询，判断是否为已删除视频
		deletedVideo, deletedErr := h.video.GetVideoByIDUnscoped(c.Request.Context(), uint(id))
		if deletedErr == nil && deletedVideo.DeletedAt.Valid {
			response.Success(c, gin.H{
				"id":         deletedVideo.ID,
				"is_deleted": true,
				"title":      deletedVideo.Title,
				"author_id":  deletedVideo.AuthorID,
			})
			return
		}
		response.NotFound(c, "video not found")
		return
	}

	tags, _ := h.video.GetVideoTags(c.Request.Context(), uint(id))

	// Get author info
	author, _ := h.video.GetAccountByID(c.Request.Context(), video.AuthorID)
	var authorData interface{}
	if author != nil {
		authorData = gin.H{
			"id":             author.ID,
			"username":       author.Username,
			"avatar_url":     author.AvatarURL,
			"bio":            author.Bio,
			"follower_count": author.FollowerCount,
		}
	}

	userID := middleware.GetUserID(c)
	isLiked := false
	isFollowing := false

	var playToken string
	if video != nil {
		ip := c.ClientIP()
		playToken, _ = h.video.GeneratePlayToken(c.Request.Context(), video.ID, userID, ip)
	}

	response.Success(c, gin.H{
		"id":                  video.ID,
		"author":              authorData,
		"title":               video.Title,
		"description":         video.Description,
		"play_url":            video.PlayURL,
		"cover_url":           video.CoverURL,
		"duration":            video.Duration,
		"width":               video.Width,
		"height":              video.Height,
		"create_time":         video.CreateTime,
		"likes_count":         video.LikesCount,
		"comment_count":       video.CommentCount,
		"view_count":          video.ViewCount,
		"popularity":          video.Popularity,
		"tags":                tags,
		"is_liked":            isLiked,
		"is_following_author": isFollowing,
		"play_token":          playToken,
	})
}

// @Summary 更新视频信息
// @Description 更新视频的标题、描述、封面或标签
// @Tags 视频
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param id path int true "视频ID"
// @Param body body UpdateVideoRequest true "更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/v1/videos/{id} [put]
func (h *VideoHandler) UpdateVideo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	var req struct {
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		CoverURL    *string  `json:"cover_url"`
		Tags        []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.CoverURL != nil {
		updates["cover_url"] = *req.CoverURL
	}

	if err := h.video.UpdateVideo(c.Request.Context(), uint(id), userID, updates); err != nil {
		if err.Error() == "forbidden" {
			response.Forbidden(c, "not the author")
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// @Summary 删除视频
// @Description 软删除视频，仅作者可删除
// @Tags 视频
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/v1/videos/{id} [delete]
func (h *VideoHandler) DeleteVideo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	if err := h.video.DeleteVideo(c.Request.Context(), uint(id), userID); err != nil {
		if err.Error() == "forbidden" {
			response.Forbidden(c, "not the author")
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// @Summary 上报播放记录
// @Description 验证 play_token，解析 user_id/ip，半小时限流后累加播放量
// @Tags 视频
// @Accept json
// @Produce json
// @Param body body RecordViewRequest true "播放记录"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/metrics/view [post]
func (h *VideoHandler) RecordView(c *gin.Context) {
	var req struct {
		PlayToken string `json:"play_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	claims, err := h.video.ValidatePlayToken(req.PlayToken)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	userID := claims.UserID

	h.video.RecordView(c.Request.Context(), claims.VideoID, userID)

	response.Success(c, gin.H{
		"user_id":  userID,
		"client_ip": claims.ClientIP,
		"video_id": claims.VideoID,
	})
}

// @Summary 初始化切片上传
// @Description 创建切片上传会话，返回 upload_id
// @Tags 视频
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param body body object true "上传参数 {filename, file_size, chunk_size}"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/videos/upload/init [post]
func (h *VideoHandler) InitChunkedUpload(c *gin.Context) {
	var req struct {
		Filename  string `json:"filename" binding:"required,min=1,max=255"`
		FileSize  int64  `json:"file_size" binding:"required,min=1"`
		ChunkSize int64  `json:"chunk_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	uploadID, err := h.video.InitChunkedUpload(c.Request.Context(), req.Filename, req.FileSize, req.ChunkSize)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Created(c, gin.H{
		"upload_id": uploadID,
	})
}

// @Summary 上传分片
// @Description 上传视频的一个分片
// @Tags 视频
// @Security OAuth2Password
// @Accept multipart/form-data
// @Produce json
// @Param upload_id formData string true "上传会话ID"
// @Param chunk_index formData int true "分片序号"
// @Param file formData file true "分片文件"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/videos/upload/chunk [post]
func (h *VideoHandler) UploadChunk(c *gin.Context) {
	uploadID := c.PostForm("upload_id")
	chunkIndexStr := c.PostForm("chunk_index")
	if uploadID == "" || chunkIndexStr == "" {
		response.BadRequest(c, "missing upload_id or chunk_index")
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		response.BadRequest(c, "invalid chunk_index")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "missing chunk file")
		return
	}

	// Validate chunk size matches session expectation
	session, err := h.video.GetUploadStatus(c.Request.Context(), uploadID)
	if err == nil && file.Size > session.ChunkSize {
		response.BadRequest(c, fmt.Sprintf("chunk size exceeds limit: %d > %d", file.Size, session.ChunkSize))
		return
	}

	src, err := file.Open()
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	defer src.Close()

	if err := h.video.UploadChunk(c.Request.Context(), uploadID, chunkIndex, src); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Created(c, gin.H{"chunk_index": chunkIndex})
}

// @Summary 查询上传进度
// @Description 查询切片上传会话的当前状态（用于断点续传）
// @Tags 视频
// @Security OAuth2Password
// @Produce json
// @Param upload_id path string true "上传会话ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/videos/upload/status/{upload_id} [get]
func (h *VideoHandler) GetUploadStatus(c *gin.Context) {
	uploadID := c.Param("upload_id")
	session, err := h.video.GetUploadStatus(c.Request.Context(), uploadID)
	if err != nil {
		response.NotFound(c, "upload session not found")
		return
	}

	response.Success(c, session)
}

// @Summary 完成切片上传
// @Description 合并所有分片，返回视频播放URL和元信息
// @Tags 视频
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param body body object true "{upload_id}"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/videos/upload/complete [post]
func (h *VideoHandler) CompleteChunkedUpload(c *gin.Context) {
	var req struct {
		UploadID string `json:"upload_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	url, meta, err := h.video.CompleteChunkedUpload(c.Request.Context(), req.UploadID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Created(c, gin.H{
		"play_url":    url,
		"duration":    meta.Duration,
		"width":       meta.Width,
		"height":      meta.Height,
		"is_vertical": meta.IsVertical,
	})
}

// Request types
type (
	PublishVideoRequest struct {
		Title       string   `json:"title" binding:"required"`
		Description string   `json:"description"`
		PlayURL     string   `json:"play_url" binding:"required"`
		CoverURL    string   `json:"cover_url" binding:"required"`
		Tags        []string `json:"tags"`
	}
	UpdateVideoRequest struct {
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		CoverURL    *string  `json:"cover_url"`
		Tags        []string `json:"tags"`
	}
)
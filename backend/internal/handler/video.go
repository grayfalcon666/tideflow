package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
	"tideflow/internal/service"
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

	url, err := h.video.UploadVideo(c.Request.Context(), file)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Created(c, gin.H{"play_url": url})
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

	videoID, err := h.video.PublishVideo(c.Request.Context(), userID, username, req.Title, req.Description, req.PlayURL, req.CoverURL, req.Tags)
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
	_ = userID
	isLiked := false
	isFollowing := false

	response.Success(c, gin.H{
		"id":                  video.ID,
		"author":              authorData,
		"title":               video.Title,
		"description":         video.Description,
		"play_url":            video.PlayURL,
		"cover_url":           video.CoverURL,
		"create_time":         video.CreateTime,
		"likes_count":         video.LikesCount,
		"popularity":          video.Popularity,
		"tags":                tags,
		"is_liked":            isLiked,
		"is_following_author": isFollowing,
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
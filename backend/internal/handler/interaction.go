package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type InteractionHandler struct {
	interaction *service.InteractionService
}

func NewInteractionHandler(interaction *service.InteractionService) *InteractionHandler {
	return &InteractionHandler{interaction: interaction}
}

// @Summary 点赞视频
// @Description 给指定视频点赞，幂等操作
// @Tags 互动
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/videos/{id}/like [post]
func (h *InteractionHandler) LikeVideo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	if err := h.interaction.LikeVideo(c.Request.Context(), userID, uint(videoID)); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// @Summary 取消点赞
// @Description 取消对指定视频的点赞
// @Tags 互动
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/videos/{id}/like [delete]
func (h *InteractionHandler) UnlikeVideo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	if err := h.interaction.UnlikeVideo(c.Request.Context(), userID, uint(videoID)); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// @Summary 检查是否点赞
// @Description 检查当前用户是否点赞了指定视频
// @Tags 互动
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/videos/{id}/like [get]
func (h *InteractionHandler) IsLiked(c *gin.Context) {
	userID := middleware.GetUserID(c)
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	isLiked, _ := h.interaction.IsLiked(c.Request.Context(), userID, uint(videoID))
	response.Success(c, gin.H{"is_liked": isLiked})
}

// @Summary 我点赞过的视频
// @Description 获取当前用户点赞过的视频列表
// @Tags 互动
// @Security OAuth2Password
// @Produce json
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Failure 401 {object} response.Response
// @Router /api/v1/likes/mine [get]
func (h *InteractionHandler) GetLikedVideos(c *gin.Context) {
	userID := middleware.GetUserID(c)
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	videos, nextCursor, hasMore, err := h.interaction.GetLikedVideos(c.Request.Context(), userID, cursor, limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, response.PageResponse{
		Items:      videos,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

// @Summary 发表评论
// @Description 对视频发表评论或回复
// @Tags 互动
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param id path int true "视频ID"
// @Param body body PublishCommentRequest true "评论内容"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/videos/{id}/comments [post]
func (h *InteractionHandler) PublishComment(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username, _ := c.Get("username")
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	var req struct {
		Content  string `json:"content" binding:"required"`
		ParentID uint   `json:"parent_id"`
		RootID   uint   `json:"root_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	uname := ""
	if u, ok := username.(string); ok {
		uname = u
	}

	commentID, err := h.interaction.PublishComment(c.Request.Context(), uint(videoID), userID, uname, req.Content, req.ParentID, req.RootID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Created(c, gin.H{"comment_id": commentID})
}

// @Summary 删除评论
// @Description 删除指定评论，仅评论作者可删除
// @Tags 互动
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Param comment_id path int true "评论ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/videos/{id}/comments/{comment_id} [delete]
func (h *InteractionHandler) DeleteComment(c *gin.Context) {
	userID := middleware.GetUserID(c)
	videoIDStr := c.Param("id")
	_, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}
	commentIDStr := c.Param("comment_id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid comment id")
		return
	}

	if err := h.interaction.DeleteComment(c.Request.Context(), uint(commentID), userID); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// @Summary 评论列表
// @Description 获取视频的评论列表
// @Tags 互动
// @Produce json
// @Param id path int true "视频ID"
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Param root_id query int false "根评论ID"
// @Success 200 {object} response.PageResponse
// @Failure 400 {object} response.Response
// @Router /api/v1/videos/{id}/comments [get]
func (h *InteractionHandler) GetComments(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	var rootID uint
	if rid := c.Query("root_id"); rid != "" {
		if parsed, err := strconv.ParseUint(rid, 10, 64); err == nil {
			rootID = uint(parsed)
		}
	}

	comments, nextCursor, hasMore, err := h.interaction.GetComments(c.Request.Context(), uint(videoID), rootID, cursor, limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, response.PageResponse{
		Items:      comments,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

// Request types
type (
	PublishCommentRequest struct {
		Content  string `json:"content" binding:"required"`
		ParentID uint   `json:"parent_id"`
		RootID   uint   `json:"root_id"`
	}
)
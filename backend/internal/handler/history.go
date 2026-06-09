package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type HistoryHandler struct {
	history *service.HistoryService
}

func NewHistoryHandler(history *service.HistoryService) *HistoryHandler {
	return &HistoryHandler{history: history}
}

// @Summary 记录观看历史
// @Description 记录观看历史（30分钟去重）
// @Tags 视频
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param body body object true "{video_id}"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/history/record [post]
func (h *HistoryHandler) RecordHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		VideoID uint `json:"video_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.history.RecordHistory(c.Request.Context(), userID, req.VideoID); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// @Summary 获取观看历史
// @Description 分页获取当前用户的观看历史
// @Tags 视频
// @Security OAuth2Password
// @Produce json
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Failure 401 {object} response.Response
// @Router /api/v1/history [get]
func (h *HistoryHandler) GetHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	videos, nextCursor, hasMore, err := h.history.GetHistory(c.Request.Context(), userID, cursor, limit)
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

// @Summary 删除单条观看记录
// @Description 删除指定视频的观看历史
// @Tags 视频
// @Security OAuth2Password
// @Produce json
// @Param video_id path int true "视频ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/history/{video_id} [delete]
func (h *HistoryHandler) DeleteHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	videoIDStr := c.Param("video_id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	if err := h.history.DeleteHistory(c.Request.Context(), userID, uint(videoID)); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// @Summary 清空观看历史
// @Description 清空当前用户的所有观看历史
// @Tags 视频
// @Security OAuth2Password
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/history [delete]
func (h *HistoryHandler) ClearHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.history.ClearHistory(c.Request.Context(), userID); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

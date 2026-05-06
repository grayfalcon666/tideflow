package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type NotificationHandler struct {
	notif *service.NotificationService
}

func NewNotificationHandler(notif *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notif: notif}
}

// @Summary 通知列表
// @Description 获取当前用户的通知列表
// @Tags 通知
// @Security OAuth2Password
// @Produce json
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Failure 401 {object} response.Response
// @Router /api/v1/notifications [get]
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	notifs, nextCursor, hasMore, err := h.notif.GetNotifications(c.Request.Context(), userID, cursor, limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, response.PageResponse{
		Items:      notifs,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

// @Summary 标记通知已读
// @Description 标记通知为已读，传入IDs只标记指定通知，不传或空数组标记全部
// @Tags 通知
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param body body MarkReadRequest false "通知ID列表"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/notifications/read [put]
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		IDs []uint `json:"ids"`
	}
	c.ShouldBindJSON(&req)

	if err := h.notif.MarkRead(c.Request.Context(), userID, req.IDs); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// @Summary 未读通知数
// @Description 获取当前用户未读通知的数量
// @Tags 通知
// @Security OAuth2Password
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/notifications/unread-count [get]
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	count, err := h.notif.CountUnread(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"count": count})
}

// Request types
type (
	MarkReadRequest struct {
		IDs []uint `json:"ids"`
	}
)
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type MessageHandler struct {
	msg *service.MessageService
}

func NewMessageHandler(msg *service.MessageService) *MessageHandler {
	return &MessageHandler{msg: msg}
}

// @Summary 发送私信
// @Description 向指定用户发送私信
// @Tags 私信
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body SendMessageRequest true "私信内容"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/messages [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		ToID    uint   `json:"to_id" binding:"required"`
		Content string `json:"content" binding:"required,max=500"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	id, err := h.msg.SendMessage(c.Request.Context(), userID, req.ToID, req.Content)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Created(c, gin.H{"message_id": id})
}

// @Summary 会话列表
// @Description 获取当前用户的会话列表
// @Tags 私信
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/messages/conversations [get]
func (h *MessageHandler) GetConversations(c *gin.Context) {
	userID := middleware.GetUserID(c)
	convs, err := h.msg.GetConversations(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"items": convs})
}

// @Summary 获取消息记录
// @Description 获取与指定用户的聊天记录
// @Tags 私信
// @Security BearerAuth
// @Produce json
// @Param user_id path int true "用户ID"
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/messages/conversations/{user_id} [get]
func (h *MessageHandler) GetMessages(c *gin.Context) {
	userID := middleware.GetUserID(c)
	otherIDStr := c.Param("user_id")
	otherID, err := strconv.ParseUint(otherIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	msgs, nextCursor, hasMore, err := h.msg.GetMessages(c.Request.Context(), userID, uint(otherID), cursor, limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, response.PageResponse{
		Items:      msgs,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

// @Summary 标记会话已读
// @Description 标记与指定用户的会话为已读
// @Tags 私信
// @Security BearerAuth
// @Produce json
// @Param user_id path int true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/messages/conversations/{user_id}/read [put]
func (h *MessageHandler) MarkRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	otherIDStr := c.Param("user_id")
	otherID, err := strconv.ParseUint(otherIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	if err := h.msg.MarkRead(c.Request.Context(), userID, uint(otherID)); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// Request types
type (
	SendMessageRequest struct {
		ToID    uint   `json:"to_id" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
)
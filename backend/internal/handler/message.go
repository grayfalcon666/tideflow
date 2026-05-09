package handler

import (
	"encoding/json"
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
// @Security OAuth2Password
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

	id, createdAt, err := h.msg.SendMessage(c.Request.Context(), userID, req.ToID, req.Content)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Created(c, gin.H{
		"id":         id,
		"from_id":    userID,
		"to_id":      req.ToID,
		"content":    req.Content,
		"created_at": createdAt,
		"is_read":    false,
	})
}

// @Summary 会话列表
// @Description 获取当前用户的会话列表
// @Tags 私信
// @Security OAuth2Password
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
// @Security OAuth2Password
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

// @Summary 分享视频给好友
// @Description 将视频预览消息批量发送给指定好友
// @Tags 私信
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param body body object true "视频分享请求"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/messages/share-video [post]
func (h *MessageHandler) ShareVideo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		ToIDs      []uint `json:"to_ids" binding:"required,min=1,max=20"`
		VideoID    uint   `json:"video_id" binding:"required"`
		Title      string `json:"title" binding:"required,max=255"`
		CoverURL   string `json:"cover_url" binding:"required,max=512"`
		AuthorName string `json:"author_name" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	contentBytes, err := json.Marshal(gin.H{
		"video_id":    req.VideoID,
		"title":       req.Title,
		"cover_url":   req.CoverURL,
		"author_name": req.AuthorName,
	})
	if err != nil {
		response.InternalServerError(c, "failed to marshal video share payload")
		return
	}

	msgs, err := h.msg.ShareVideo(c.Request.Context(), userID, req.ToIDs, string(contentBytes))
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Created(c, gin.H{"messages": msgs})
}

// @Summary 标记会话已读
// @Description 标记与指定用户的会话为已读
// @Tags 私信
// @Security OAuth2Password
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
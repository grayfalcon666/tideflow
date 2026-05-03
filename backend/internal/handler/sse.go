package handler

import (
	"io"

	"github.com/gin-gonic/gin"

	"tideflow/internal/mq"
)

type SSEHandler struct {
	hub *mq.SSEHub
}

func NewSSEHandler(hub *mq.SSEHub) *SSEHandler {
	return &SSEHandler{hub: hub}
}

// @Summary SSE实时通知
// @Description 建立SSE连接，接收实时通知推送
// @Tags 通知
// @Security BearerAuth
// @Produce text/event-stream
// @Param token query string true "访问令牌"
// @Success 200 {string} string "SSE事件流"
// @Failure 401 {object} response.Response
// @Router /api/v1/notifications/stream [get]
func (h *SSEHandler) Stream(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"code": 401, "message": "unauthorized"})
		return
	}

	ch := h.hub.Register(userID.(uint))

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case data, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("message", data)
			return true
		case <-c.Request.Context().Done():
			h.hub.Unregister(userID.(uint))
			return false
		}
	})
}

type TagHandler struct {
}

func NewTagHandler() *TagHandler {
	return &TagHandler{}
}

// @Summary 热门标签
// @Description 获取当前热门标签列表
// @Tags 标签
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/tags/hot [get]
func (h *TagHandler) GetHotTags(c *gin.Context) {
	tags := []string{"搞笑", "音乐", "游戏"}
	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"tags": tags,
		},
	})
}
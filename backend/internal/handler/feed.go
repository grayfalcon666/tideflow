package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type FeedHandler struct {
	feed *service.FeedService
}

func NewFeedHandler(feed *service.FeedService) *FeedHandler {
	return &FeedHandler{feed: feed}
}

// @Summary 全局最新Feed
// @Description 获取全局最新视频列表，按时间倒序
// @Tags 时间线
// @Produce json
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Router /api/v1/feed/latest [get]
func (h *FeedHandler) ListLatest(c *gin.Context) {
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	videos, nextCursor, hasMore, err := h.feed.ListLatest(c.Request.Context(), cursor, limit)
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

// @Summary 热门榜单
// @Description 获取热门视频列表，按热度降序
// @Tags 时间线
// @Produce json
// @Param cursor query string false "偏移量游标" default(0)
// @Param limit query int false "每页数量" default(20)
// @Param window query string false "时间窗口" default(1h) enum(1m,5m,15m,1h,6h)
// @Success 200 {object} response.PageResponse
// @Router /api/v1/feed/popular [get]
func (h *FeedHandler) ListPopular(c *gin.Context) {
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	window := c.DefaultQuery("window", "1h")

	// window 参数白名单校验
	validWindows := map[string]bool{"1m": true, "5m": true, "15m": true, "1h": true, "6h": true}
	if !validWindows[window] {
		response.BadRequest(c, "invalid window, must be one of: 1m, 5m, 15m, 1h, 6h")
		return
	}

	offset := 0
	if cursor != "" {
		if parsed, err := strconv.Atoi(cursor); err == nil {
			offset = parsed
		}
	}

	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	videos, nextCursor, hasMore, err := h.feed.ListPopular(c.Request.Context(), offset, limit, window)
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

// @Summary 关注流
// @Description 获取当前用户关注的人的发布视频
// @Tags 时间线
// @Security OAuth2Password
// @Produce json
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Failure 401 {object} response.Response
// @Router /api/v1/feed/following [get]
func (h *FeedHandler) ListByFollowing(c *gin.Context) {
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	userID := middleware.GetUserID(c)

	videos, nextCursor, hasMore, err := h.feed.ListByFollowing(c.Request.Context(), userID, cursor, limit)
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

// @Summary 按标签搜索视频
// @Description 通过标签搜索视频列表
// @Tags 时间线
// @Produce json
// @Param tag query string true "标签名"
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Failure 400 {object} response.Response
// @Router /api/v1/feed/tag [get]
func (h *FeedHandler) ListByTag(c *gin.Context) {
	tag := c.Query("tag")
	if tag == "" {
		response.BadRequest(c, "tag is required")
		return
	}

	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	videos, nextCursor, hasMore, err := h.feed.ListByTag(c.Request.Context(), tag, cursor, limit)
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
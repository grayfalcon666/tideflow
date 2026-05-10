package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type SearchHandler struct {
	search *service.SearchService
}

func NewSearchHandler(search *service.SearchService) *SearchHandler {
	return &SearchHandler{search: search}
}

// @Summary 搜索视频
// @Description 通过关键词搜索视频，支持按热度或时间排序
// @Tags 搜索
// @Produce json
// @Param q query string true "搜索关键词"
// @Param sort_by query string false "排序字段：popularity（默认）/ create_time"
// @Param order query string false "排序方向：desc（默认）/ asc"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} response.SearchResponse
// @Failure 400 {object} response.Response
// @Security OAuth2Password
// @Router /api/v1/search/videos [get]
func (h *SearchHandler) SearchVideos(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.BadRequest(c, "q (query) is required")
		return
	}

	sortBy := c.DefaultQuery("sort_by", "popularity")
	if sortBy != "popularity" && sortBy != "create_time" {
		response.BadRequest(c, "sort_by must be 'popularity' or 'create_time'")
		return
	}

	order := c.DefaultQuery("order", "desc")
	if order != "asc" && order != "desc" {
		response.BadRequest(c, "order must be 'asc' or 'desc'")
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	sizeStr := c.DefaultQuery("size", "20")
	size := 20
	if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 50 {
		size = s
	}

	from := (page - 1) * size

	// 1. ES 搜索获取 video_id 列表
	ids, total, err := h.search.SearchVideos(c.Request.Context(), query, sortBy, order, from, size)
	if err != nil {
		response.InternalServerError(c, "search failed: "+err.Error())
		return
	}

	// 2. 三级缓存补全视频实体
	videos, err := h.search.EnrichVideoIDs(c.Request.Context(), ids)
	if err != nil {
		response.InternalServerError(c, "failed to fetch video details: "+err.Error())
		return
	}

	// 3. 补充作者信息（头像、大V标记）
	h.search.EnrichWithAccountData(c.Request.Context(), videos)

	hasMore := int64(from+len(videos)) < total

	var nextCursor *string
	if hasMore && len(videos) > 0 {
		nc := strconv.Itoa(page + 1)
		nextCursor = &nc
	}

	response.Success(c, response.SearchResponse{
		Items:      videos,
		Total:      total,
		Page:       page,
		Size:       size,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	})
}
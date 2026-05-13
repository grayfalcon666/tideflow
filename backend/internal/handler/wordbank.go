package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type WordbankHandler struct {
	wb *service.WordbankService
}

func NewWordbankHandler(wb *service.WordbankService) *WordbankHandler {
	return &WordbankHandler{wb: wb}
}

// @Summary 获取视频词库
// @Description 获取视频的英语单词词库（含释义、音标、语境例句）
// @Tags 词库
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Success 200 {object} service.WordbankResponse
// @Failure 202 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/videos/{id}/wordbank [get]
func (h *WordbankHandler) GetWordbank(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video_id")
		return
	}

	slog.Info("GET /wordbank", "video_id", id)
	resp, err := h.wb.GetWordbank(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, service.ErrWordbankProcessing) {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    202,
				"message": "wordbank still processing",
				"data":    gin.H{"status": "processing"},
			})
		} else if errors.Is(err, service.ErrWordbankFailed) {
			response.NotFound(c, "wordbank generation failed or not available")
		} else if errors.Is(err, service.ErrWordbankNotFound) {
			response.NotFound(c, "wordbank not found")
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	slog.Info("GET /wordbank 完成", "video_id", id, "size", resp.Size)
	response.Success(c, resp)
}

// @Summary 导出视频词库
// @Description 导出视频词库为 JSON 文件下载
// @Tags 词库
// @Security OAuth2Password
// @Produce application/json
// @Param id path int true "视频ID"
// @Success 200 {file} json
// @Failure 404 {object} response.Response
// @Router /api/v1/videos/{id}/wordbank/export [get]
func (h *WordbankHandler) ExportWordbank(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video_id")
		return
	}

	data, filename, err := h.wb.ExportWordbank(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, service.ErrWordbankProcessing) {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    202,
				"message": "wordbank still processing",
				"data":    gin.H{"status": "processing"},
			})
		} else if errors.Is(err, service.ErrWordbankFailed) {
			response.NotFound(c, "wordbank generation failed or not available")
		} else if errors.Is(err, service.ErrWordbankNotFound) {
			response.NotFound(c, "wordbank not found")
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Data(http.StatusOK, "application/json", data)
}

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

type LearningHandler struct {
	svc *service.LearningService
}

func NewLearningHandler(svc *service.LearningService) *LearningHandler {
	return &LearningHandler{svc: svc}
}

// @Summary 获取考纲词表列表
// @Description 返回所有预置的考纲词表（如四级、六级等）
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/learn/lists [get]
func (h *LearningHandler) GetLists(c *gin.Context) {
	lists, err := h.svc.GetLists()
	if err != nil {
		if errors.Is(err, service.ErrVocabNotInit) {
			response.InternalServerError(c, "vocab lists not initialized")
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}
	response.Success(c, lists)
}

// @Summary 获取视频中属于指定词表的单词
// @Description 计算视频词库与指定考纲词表的交集，返回可用于学习的单词列表
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Param list_id query int true "词表ID"
// @Success 200 {object} service.LearningWordsResponse
// @Failure 404 {object} response.Response
// @Failure 412 {object} response.Response
// @Router /api/v1/videos/{id}/learn/words [get]
func (h *LearningHandler) GetLearningWords(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video_id")
		return
	}

	listID, err := strconv.ParseUint(c.Query("list_id"), 10, 64)
	if err != nil || listID == 0 {
		response.BadRequest(c, "invalid or missing list_id")
		return
	}

	slog.Info("GET /learn/words", "video_id", videoID, "list_id", listID)
	resp, err := h.svc.GetLearningWords(c.Request.Context(), uint(videoID), uint(listID))
	if err != nil {
		if errors.Is(err, service.ErrWordbankNotReady) {
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"code":    412,
				"message": "wordbank not ready",
			})
		} else if errors.Is(err, service.ErrListNotFound) {
			response.NotFound(c, "vocab list not found")
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	slog.Info("GET /learn/words 完成", "video_id", videoID, "list_name", resp.ListName, "matched", resp.Total)
	response.Success(c, resp)
}

// @Summary 获取单词语境例句
// @Description 从视频词库中获取指定单词的所有字幕语境片段
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Param word path string true "单词"
// @Success 200 {object} service.WordCaptionsResponse
// @Failure 404 {object} response.Response
// @Router /api/v1/videos/{id}/learn/word/{word}/captions [get]
func (h *LearningHandler) GetWordCaptions(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video_id")
		return
	}

	word := c.Param("word")
	if word == "" {
		response.BadRequest(c, "word is required")
		return
	}

	slog.Info("GET /learn/word/captions", "video_id", videoID, "word", word)
	resp, err := h.svc.GetWordCaptions(c.Request.Context(), uint(videoID), word)
	if err != nil {
		if errors.Is(err, service.ErrWordbankNotReady) {
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"code":    412,
				"message": "wordbank not ready",
			})
		} else if errors.Is(err, service.ErrWordNotInWb) {
			response.NotFound(c, "word not found in wordbank")
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, resp)
}

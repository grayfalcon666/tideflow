package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
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

	chunkSize, _ := strconv.Atoi(c.DefaultQuery("chunk_size", "0"))
	accountID := middleware.GetUserID(c)

	slog.Info("GET /learn/words", "video_id", videoID, "list_id", listID, "chunk_size", chunkSize)
	resp, err := h.svc.GetLearningWords(c.Request.Context(), uint(videoID), uint(listID), accountID, chunkSize)
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

// @Summary 提交学习会话
// @Description 提交一批单词的学习结果，更新视频游标、全局词汇状态和每日流水
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Param body body service.CommitLearningReq true "学习提交数据"
// @Success 200 {object} response.Response
// @Router /api/v1/learn/commit [post]
func (h *LearningHandler) CommitLearning(c *gin.Context) {
	accountID := middleware.GetUserID(c)
	if accountID == 0 {
		response.Unauthorized(c)
		return
	}

	var req service.CommitLearningReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if req.VideoID == 0 || len(req.WordsPracticed) == 0 {
		response.BadRequest(c, "video_id and words_practiced are required")
		return
	}

	if err := h.svc.CommitLearning(c.Request.Context(), accountID, req); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "commit recorded"})
}

// @Summary 获取学习习惯统计
// @Description 返回指定年份的热力图数据、打卡天数和连续打卡天数
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Param year query int false "年份" default(当前年份)
// @Success 200 {object} service.HabitStatsResp
// @Router /api/v1/learn/habit/stats [get]
func (h *LearningHandler) GetHabitStats(c *gin.Context) {
	accountID := middleware.GetUserID(c)
	if accountID == 0 {
		response.Unauthorized(c)
		return
	}

	year, _ := strconv.Atoi(c.DefaultQuery("year", "0"))

	resp, err := h.svc.GetHabitStats(c.Request.Context(), accountID, year)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, resp)
}

// @Summary 重置视频学习进度
// @Description 重置指定视频的学习游标为0，不删除全局单词状态
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Param body body service.ResetProgressReq true "视频ID"
// @Success 200 {object} response.Response
// @Router /api/v1/learn/progress/reset [post]
func (h *LearningHandler) ResetProgress(c *gin.Context) {
	accountID := middleware.GetUserID(c)
	if accountID == 0 {
		response.Unauthorized(c)
		return
	}

	var req service.ResetProgressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if req.VideoID == 0 {
		response.BadRequest(c, "video_id is required")
		return
	}

	if err := h.svc.ResetProgress(c.Request.Context(), accountID, req.VideoID); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "progress reset"})
}

// @Summary 获取今日学习的单词
// @Description 返回用户今天学习的所有单词及其星级
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/learn/today/words [get]
func (h *LearningHandler) GetTodayWords(c *gin.Context) {
	accountID := middleware.GetUserID(c)
	if accountID == 0 {
		response.Unauthorized(c)
		return
	}

	words, err := h.svc.GetTodayWords(c.Request.Context(), accountID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, words)
}

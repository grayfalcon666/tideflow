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

// @Summary 获取视频学习单词列表
// @Description 计算视频词库与考纲词表交集，返回当前批次的待学单词
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Param list_id query int true "词表ID"
// @Param chunk_size query int false "每批单词数" default(15)
// @Param mode query string true "模式: spell 或 type"
// @Success 200 {object} service.LearnWordsResp
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
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

	chunkSize, _ := strconv.Atoi(c.DefaultQuery("chunk_size", "15"))
	if chunkSize <= 0 {
		chunkSize = 15
	}
	mode := c.DefaultQuery("mode", "spell")
	if mode != "spell" && mode != "type" {
		response.BadRequest(c, "mode must be 'spell' or 'type'")
		return
	}

	accountID := middleware.GetUserID(c)
	if accountID == 0 {
		response.Unauthorized(c)
		return
	}

	slog.Info("GET /learn/words", "video_id", videoID, "list_id", listID, "chunk_size", chunkSize, "mode", mode)
	resp, err := h.svc.GetLearningWords(c.Request.Context(), uint(videoID), uint(listID), accountID, chunkSize, mode)
	if err != nil {
		if errors.Is(err, service.ErrWordbankNotReady) {
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"code":    412,
				"message": "wordbank not ready",
			})
		} else if errors.Is(err, service.ErrListNotFound) {
			response.NotFound(c, "vocab list not found")
		} else if errors.Is(err, service.ErrBatchConflict) {
			c.JSON(http.StatusConflict, gin.H{
				"code":    409,
				"message": "请先完成或放弃当前学习批次",
			})
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

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

// @Summary 提交单个单词拼写结果
// @Description 提交一个单词的拼写结果（correct/wrong），更新星级和队列
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Param body body service.CommitReq true "单词提交数据"
// @Success 200 {object} service.CommitResp
// @Failure 409 {object} response.Response
// @Router /api/v1/learn/commit [post]
func (h *LearningHandler) CommitLearning(c *gin.Context) {
	accountID := middleware.GetUserID(c)
	if accountID == 0 {
		response.Unauthorized(c)
		return
	}

	var req service.CommitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if req.Word == "" {
		response.BadRequest(c, "word is required")
		return
	}
	if req.Result != "correct" && req.Result != "wrong" {
		response.BadRequest(c, "result must be 'correct' or 'wrong'")
		return
	}

	resp, err := h.svc.CommitLearning(c.Request.Context(), accountID, req)
	if err != nil {
		if errors.Is(err, service.ErrNoActiveBatch) {
			c.JSON(http.StatusConflict, gin.H{
				"code":    409,
				"message": "no active learning batch",
			})
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, resp)
}

// @Summary 放弃当前学习批次
// @Description 删除当前活跃的学习队列和批次信息，已提交的单词状态不受影响
// @Tags 学习
// @Security OAuth2Password
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/learn/batch/abort [post]
func (h *LearningHandler) AbortBatch(c *gin.Context) {
	accountID := middleware.GetUserID(c)
	if accountID == 0 {
		response.Unauthorized(c)
		return
	}

	if err := h.svc.AbortBatch(c.Request.Context(), accountID); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, nil)
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

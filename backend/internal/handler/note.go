package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type NoteHandler struct {
	note *service.NoteService
	user *service.UserService
}

func NewNoteHandler(note *service.NoteService, user *service.UserService) *NoteHandler {
	return &NoteHandler{note: note, user: user}
}

// @Summary 创建视频笔记
// @Description 在视频指定时间点创建笔记
// @Tags 笔记
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param id path int true "视频ID"
// @Param body body CreateNoteRequest true "笔记内容"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/videos/{id}/notes [post]
func (h *NoteHandler) CreateNote(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.user.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "get user failed: "+err.Error())
		return
	}
	username := ""
	if user != nil {
		username = user.Username
	}

	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	var req struct {
		Timestamp float64 `json:"timestamp" binding:"required"`
		Content   string  `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	note, err := h.note.CreateNote(c.Request.Context(), uint(videoID), userID, username, req.Timestamp, req.Content)
	if err != nil {
		if errors.Is(err, service.ErrNoteLimitExceeded) {
			response.BadRequest(c, err.Error())
		} else if errors.Is(err, service.ErrNoteNotFound) {
			response.NotFound(c, err.Error())
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Created(c, note)
}

// @Summary 获取视频笔记列表
// @Description 获取视频的时间轴笔记列表，按时间戳升序排列
// @Tags 笔记
// @Produce json
// @Param id path int true "视频ID"
// @Param cursor query string false "游标（时间戳）"
// @Param limit query int false "每页数量" default(50)
// @Success 200 {object} response.PageResponse
// @Failure 400 {object} response.Response
// @Router /api/v1/videos/{id}/notes [get]
func (h *NoteHandler) GetNotes(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid video id")
		return
	}

	cursor := c.DefaultQuery("cursor", "0")
	limitStr := c.DefaultQuery("limit", "50")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	notes, nextCursor, hasMore, err := h.note.GetNotes(c.Request.Context(), uint(videoID), cursor, limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, response.PageResponse{
		Items:      notes,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

// @Summary 删除笔记
// @Description 删除指定笔记，仅笔记作者可删除
// @Tags 笔记
// @Security OAuth2Password
// @Produce json
// @Param id path int true "视频ID"
// @Param note_id path int true "笔记ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/v1/videos/{id}/notes/{note_id} [delete]
func (h *NoteHandler) DeleteNote(c *gin.Context) {
	userID := middleware.GetUserID(c)

	noteIDStr := c.Param("note_id")
	noteID, err := strconv.ParseUint(noteIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid note id")
		return
	}

	if err := h.note.DeleteNote(c.Request.Context(), uint(noteID), userID); err != nil {
		if errors.Is(err, service.ErrNoteNotFound) {
			response.NotFound(c, err.Error())
		} else if errors.Is(err, service.ErrNotNoteAuthor) {
			response.Forbidden(c, err.Error())
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

type CreateNoteRequest struct {
	Timestamp float64 `json:"timestamp" binding:"required"`
	Content   string  `json:"content" binding:"required"`
}

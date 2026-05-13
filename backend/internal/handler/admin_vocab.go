package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type AdminVocabHandler struct {
	svc         *service.LearningService
	vocabListDir string
}

func NewAdminVocabHandler(svc *service.LearningService, vocabListDir string) *AdminVocabHandler {
	return &AdminVocabHandler{svc: svc, vocabListDir: vocabListDir}
}

// @Summary 导入考纲词表
// @Description 扫描 VOCAB_LISTS_DIR 下的 .txt 文件并导入到 MySQL，然后重新加载内存缓存
// @Tags 管理
// @Security OAuth2Password
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/admin/vocab/import [post]
func (h *AdminVocabHandler) ImportVocabLists(c *gin.Context) {
	if err := h.svc.ImportVocabLists(c.Request.Context(), h.vocabListDir); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "vocab lists imported and reloaded",
	})
}

// @Summary 重载考纲词表缓存
// @Description 从 MySQL 重新加载考纲词表到进程内存
// @Tags 管理
// @Security OAuth2Password
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/admin/vocab/reload [post]
func (h *AdminVocabHandler) ReloadVocab(c *gin.Context) {
	if err := h.svc.Reload(c.Request.Context()); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "vocab cache reloaded",
	})
}

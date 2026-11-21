package response

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// InternalErrorMessage 是 5xx 返回给客户端的统一通用文案，内部错误细节只进日志。
const InternalErrorMessage = "服务器内部错误，请稍后重试"

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, 400, message)
}

func Unauthorized(c *gin.Context) {
	Error(c, http.StatusUnauthorized, 401, "unauthorized")
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, 403, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, 404, message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, 409, message)
}

// PreconditionFailed 用于业务状态不满足前置条件的场景（如学习批次进行中）。
func PreconditionFailed(c *gin.Context, message string) {
	Error(c, http.StatusPreconditionFailed, 412, message)
}

// Accepted 用于异步处理中的轮询态响应，客户端按 code 非 0 处理。
func Accepted(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusAccepted, Response{
		Code:    202,
		Message: message,
		Data:    data,
	})
}

func TooManyRequests(c *gin.Context) {
	Error(c, http.StatusTooManyRequests, 429, "rate limit exceeded")
}

// InternalServerError 记录一条结构化错误日志，客户端只收到通用文案，不泄漏内部细节。
func InternalServerError(c *gin.Context, err error) {
	slog.Error("internal server error",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", http.StatusInternalServerError,
		"error", err)
	Error(c, http.StatusInternalServerError, 500, InternalErrorMessage)
}

type PageResponse struct {
	Items      interface{} `json:"items"`
	NextCursor *string     `json:"next_cursor"`
	HasMore    bool        `json:"has_more"`
}

type SearchResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Size       int         `json:"size"`
	HasMore    bool        `json:"has_more"`
	NextCursor *string     `json:"next_cursor,omitempty"`
}

type TokenResponse struct {
	AccountID    uint   `json:"account_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

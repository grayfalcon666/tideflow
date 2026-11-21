package middleware

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"tideflow/pkg/response"
)

// Recovery 捕获 handler panic，返回 JSON 通用 500 并记录结构化日志，
// 替代 gin.Default() 自带的纯文本 Recovery。
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		response.InternalServerError(c, errors.New(fmt.Sprint(recovered)))
	})
}

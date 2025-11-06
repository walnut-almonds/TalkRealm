package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

// Logger 日誌中介軟體
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		logger.Info("HTTP Request",
			"status", c.Writer.Status(),
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"ip", c.ClientIP(),
			"user-agent", c.Request.UserAgent(),
			"latency", latency,
			"error", c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}

// CORS 跨域資源共享中介軟體
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Auth JWT 認證中介軟體
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 實作 JWT 驗證邏輯
		// 1. 從 Authorization header 取得 token
		// 2. 驗證 token
		// 3. 將使用者資訊放入 context

		// 目前先放行
		c.Next()
	}
}

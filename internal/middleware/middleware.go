package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"github.com/walnut-almonds/talkrealm/pkg/auth"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

// roleLevel 定義角色層級，數字越大代表權限越高
var roleLevel = map[string]int{
	"member":    1,
	"moderator": 2,
	"admin":     3,
	"owner":     4,
}

// HasMinRole 判斷 userRole 是否達到 minRole 的要求
func HasMinRole(userRole, minRole string) bool {
	return roleLevel[userRole] >= roleLevel[minRole]
}

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
		c.Writer.Header().
			Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().
			Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// AuthMiddleware JWT 認證中間件
func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 從 Authorization header 取得 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			c.Abort()

			return
		}

		// 檢查 Bearer 前綴
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()

			return
		}

		tokenString := parts[1]

		// 驗證 token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			c.Abort()

			return
		}

		// 將使用者資訊存入 context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		c.Next()
	}
}

// Auth 舊版相容 - 使用預設配置
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

// RequireGuildRole 驗證當前使用者在社群中的角色是否達到最低要求。
// 社群 ID 從 URL 參數 `:id` 取得。
// 角色層級（由低到高）：member < moderator < admin < owner
func RequireGuildRole(
	minRole string,
	guildMemberRepo repository.GuildMemberRepository,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()

			return
		}

		guildIDStr := c.Param("id")

		guildID, err := strconv.ParseUint(guildIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild ID"})
			c.Abort()

			return
		}

		member, err := guildMemberRepo.GetMember(uint(guildID), userID)
		if err != nil || member == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this guild"})
			c.Abort()

			return
		}

		if !HasMinRole(member.Role, minRole) {
			c.JSON(
				http.StatusForbidden,
				gin.H{"error": "insufficient permissions, required role: " + minRole},
			)
			c.Abort()

			return
		}

		// 將 guild member 存入 context，供下游 handler 直接使用
		c.Set("guild_member", member)
		c.Next()
	}
}

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

// MessageRateLimit 訊息發送速率限制中介軟體
// 每位使用者每秒最多允許 maxMsg 則訊息，超過時回傳 429。
func MessageRateLimit(rdb *goredis.Client, maxMsg int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%v:msg", userID)
		ctx := context.Background()

		pipe := rdb.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Second)
		if _, err := pipe.Exec(ctx); err != nil {
			// Redis 錯誤時放行，避免影響正常服務
			c.Next()
			return
		}

		if incr.Val() > int64(maxMsg) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded; please slow down",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

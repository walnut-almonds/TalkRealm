package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	pkgredis "github.com/walnut-almonds/talkrealm/pkg/redis"
)

// MessageRateLimit 訊息發送速率限制中介軟體
// 每位使用者每秒最多允許 maxMsg 則訊息，超過時回傳 429。rdb 為 nil 時不限流。
func MessageRateLimit(rdb *goredis.Client, maxMsg int) gin.HandlerFunc {
	if rdb == nil {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%v:msg", userID)
		if !pkgredis.AllowFixedWindow(rdb, key, maxMsg, time.Second) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded; please slow down",
			})
			c.Abort()

			return
		}

		c.Next()
	}
}

// IPRateLimit 以來源 IP 限制未認證端點（登入、註冊）的呼叫頻率，
// 讓密碼暴力破解與大量開帳號在單一來源下不可行。rdb 為 nil 時不限流。
func IPRateLimit(
	rdb *goredis.Client,
	name string,
	maxHits int,
	window time.Duration,
) gin.HandlerFunc {
	if rdb == nil {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		key := fmt.Sprintf("ratelimit:ip:%s:%s", name, c.ClientIP())
		if !pkgredis.AllowFixedWindow(rdb, key, maxHits, window) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many attempts; please try again later",
			})
			c.Abort()

			return
		}

		c.Next()
	}
}

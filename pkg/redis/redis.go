package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/walnut-almonds/talkrealm/pkg/config"
)

var (
	client *goredis.Client
	once   sync.Once
)

// NewClient 初始化並回傳 Redis client（singleton）
func NewClient(cfg *config.RedisConfig) (*goredis.Client, error) {
	var initErr error

	once.Do(func() {
		c := goredis.NewClient(&goredis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Password: cfg.Password,
			DB:       cfg.DB,
		})

		ctx := context.Background()
		if _, err := c.Ping(ctx).Result(); err != nil {
			initErr = fmt.Errorf("redis ping failed: %w", err)
			return
		}

		client = c
	})

	if initErr != nil {
		return nil, initErr
	}

	return client, nil
}

// GetClient 回傳已初始化的 Redis client；未初始化時 panic
func GetClient() *goredis.Client {
	if client == nil {
		panic("redis client not initialized; call NewClient first")
	}

	return client
}

// AllowFixedWindow 對 key 做「每 window 最多 maxHits 次」的固定視窗計數。
// 用 EXPIRE NX 而非無條件 EXPIRE：後者會讓持續發送的人不斷把 TTL 往後推，
// 視窗永不重置、計數只增不減。NX 也比「只在 hits==1 時設 TTL」安全——
// 那種寫法只要那一次 EXPIRE 掉了（Redis 抖動、行程剛好掛掉），key 就永久沒有
// TTL，該使用者／IP 會被永遠鎖死。兩個指令走同一個 pipeline，維持單次往返。
// Redis 故障時放行，不因附屬服務擋掉正常流量。
func AllowFixedWindow(
	rdb *goredis.Client,
	key string,
	maxHits int,
	window time.Duration,
) bool {
	if rdb == nil {
		return true
	}

	ctx := context.Background()

	pipe := rdb.Pipeline()
	incr := pipe.Incr(ctx, key)

	pipe.ExpireNX(ctx, key, window)

	if _, err := pipe.Exec(ctx); err != nil {
		return true
	}

	return incr.Val() <= int64(maxHits)
}

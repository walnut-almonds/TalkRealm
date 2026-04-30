package redis

import (
	"context"
	"fmt"
	"sync"

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

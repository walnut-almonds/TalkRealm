package service

import (
	"context"
	"errors"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// LevelStore 進行中關卡的暫存（含答案，不進 DB）
type LevelStore interface {
	Get(key string) ([]byte, error) // (nil, nil) = 不存在或已過期
	Set(key string, val []byte, ttl time.Duration) error
	SetNX(key string, val []byte, ttl time.Duration) (bool, error)
}

// --- Redis 實作 ---

type redisLevelStore struct {
	rdb *goredis.Client
}

// NewRedisLevelStore 建立 Redis level store
func NewRedisLevelStore(rdb *goredis.Client) LevelStore {
	return &redisLevelStore{rdb: rdb}
}

func (s *redisLevelStore) Get(key string) ([]byte, error) {
	b, err := s.rdb.Get(context.Background(), key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return b, nil
}

func (s *redisLevelStore) Set(key string, val []byte, ttl time.Duration) error {
	return s.rdb.Set(context.Background(), key, val, ttl).Err()
}

func (s *redisLevelStore) SetNX(key string, val []byte, ttl time.Duration) (bool, error) {
	return s.rdb.SetNX(context.Background(), key, val, ttl).Result()
}

// --- in-memory 實作 ---
// ponytail: Redis 不可用時的降級（單機限定、重啟即失）；多實例部署必須用 Redis

type memoryLevelStore struct {
	mu    sync.Mutex
	items map[string]memoryItem
}

type memoryItem struct {
	val     []byte
	expires time.Time
}

// NewMemoryLevelStore 建立記憶體 level store（Redis 不可用時的 fallback 與測試用）
func NewMemoryLevelStore() LevelStore {
	return &memoryLevelStore{items: make(map[string]memoryItem)}
}

func (s *memoryLevelStore) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	it, ok := s.items[key]
	if !ok || time.Now().After(it.expires) {
		delete(s.items, key)

		return nil, nil
	}

	return it.val, nil
}

func (s *memoryLevelStore) Set(key string, val []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[key] = memoryItem{val: val, expires: time.Now().Add(ttl)}

	return nil
}

func (s *memoryLevelStore) SetNX(key string, val []byte, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if it, ok := s.items[key]; ok && time.Now().Before(it.expires) {
		return false, nil
	}

	s.items[key] = memoryItem{val: val, expires: time.Now().Add(ttl)}

	return true, nil
}

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"huoxing-search/internal/pkg/redis"
)

// CacheRepository 缓存仓储接口
type CacheRepository interface {
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, bool)
	Delete(ctx context.Context, keys ...string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	GetJSON(ctx context.Context, key string, dest interface{}) error
	SetSearchResult(ctx context.Context, keyword string, panType int, results interface{}, ttl time.Duration) error
	GetSearchResult(ctx context.Context, keyword string, panType int, dest interface{}) (bool, error)
}

type cacheRepository struct{}

// NewCacheRepository 创建缓存仓储
func NewCacheRepository() CacheRepository {
	return &cacheRepository{}
}

// Set 设置缓存(字符串)
func (r *cacheRepository) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return redis.Set(ctx, key, []byte(value), expiration)
}

// Get 获取缓存(字符串)
func (r *cacheRepository) Get(ctx context.Context, key string) (string, bool) {
	data, err := redis.Get(ctx, key)
	if err != nil {
		return "", false
	}
	return data, true
}

// SetJSON 设置缓存(JSON对象)
func (r *cacheRepository) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	return redis.Set(ctx, key, data, expiration)
}

// GetJSON 获取缓存(JSON对象)
func (r *cacheRepository) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := redis.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// Delete 删除缓存
func (r *cacheRepository) Delete(ctx context.Context, keys ...string) error {
	return redis.Del(ctx, keys...)
}

// DeletePattern 删除匹配模式的所有缓存
func (r *cacheRepository) DeletePattern(ctx context.Context, pattern string) error {
	// 使用SCAN命令查找匹配的key
	keys, err := redis.Keys(ctx, pattern)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return redis.Del(ctx, keys...)
}

// Exists 检查缓存是否存在
func (r *cacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	count, err := redis.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SetSearchResult 设置搜索结果缓存
func (r *cacheRepository) SetSearchResult(ctx context.Context, keyword string, panType int, results interface{}, ttl time.Duration) error {
	key := r.buildSearchKey(keyword, panType)
	return r.SetJSON(ctx, key, results, ttl)
}

// GetSearchResult 获取搜索结果缓存
func (r *cacheRepository) GetSearchResult(ctx context.Context, keyword string, panType int, dest interface{}) (bool, error) {
	key := r.buildSearchKey(keyword, panType)
	exists, err := r.Exists(ctx, key)
	if err != nil || !exists {
		return false, err
	}
	err = r.GetJSON(ctx, key, dest)
	if err != nil {
		return false, err
	}
	return true, nil
}

// buildSearchKey 构建搜索缓存key
func (r *cacheRepository) buildSearchKey(keyword string, panType int) string {
	return fmt.Sprintf("search:%s:%d", keyword, panType)
}
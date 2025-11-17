package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/pkg/logger"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(key string) bool
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	rate       int           // 每秒生成的令牌数
	capacity   int           // 桶容量
	tokens     map[string]int // 当前令牌数
	lastUpdate map[string]time.Time // 上次更新时间
	mu         sync.RWMutex
}

// NewTokenBucket 创建令牌桶限流器
func NewTokenBucket(rate, capacity int) *TokenBucket {
	tb := &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     make(map[string]int),
		lastUpdate: make(map[string]time.Time),
	}

	// 定期清理过期的key
	go tb.cleanup()

	return tb
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow(key string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()

	// 初始化key
	if _, exists := tb.tokens[key]; !exists {
		tb.tokens[key] = tb.capacity
		tb.lastUpdate[key] = now
	}

	// 计算应该增加的令牌数
	elapsed := now.Sub(tb.lastUpdate[key])
	tokensToAdd := int(elapsed.Seconds() * float64(tb.rate))

	if tokensToAdd > 0 {
		tb.tokens[key] = min(tb.tokens[key]+tokensToAdd, tb.capacity)
		tb.lastUpdate[key] = now
	}

	// 检查是否有可用令牌
	if tb.tokens[key] > 0 {
		tb.tokens[key]--
		return true
	}

	return false
}

// cleanup 定期清理过期的key (超过5分钟未访问)
func (tb *TokenBucket) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		tb.mu.Lock()
		now := time.Now()
		for key, lastTime := range tb.lastUpdate {
			if now.Sub(lastTime) > 5*time.Minute {
				delete(tb.tokens, key)
				delete(tb.lastUpdate, key)
			}
		}
		tb.mu.Unlock()
	}
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	if !cfg.RateLimit.Enabled {
		// 限流未启用,直接放行
		return func(c *gin.Context) {
			c.Next()
		}
	}

	rate := int(cfg.RateLimit.Rate)
	if rate == 0 {
		rate = int(cfg.RateLimit.RequestsPerSecond)
	}
	limiter := NewTokenBucket(rate, cfg.RateLimit.Burst)

	return func(c *gin.Context) {
		// 使用IP作为限流key
		key := c.ClientIP()

		// 如果用户已登录,使用用户ID作为key
		if userID, exists := c.Get("user_id"); exists {
			key = fmt.Sprintf("user_%v", userID)
		}

		if !limiter.Allow(key) {
			logger.Warn("请求被限流",
				zap.String("ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path),
				zap.String("key", key),
			)

			c.JSON(http.StatusTooManyRequests, model.Response{
				Code:    http.StatusTooManyRequests,
				Message: "请求过于频繁,请稍后再试",
				Data:    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRateLimitMiddleware IP限流中间件 (更严格的IP限流)
func IPRateLimitMiddleware(rate, burst int) gin.HandlerFunc {
	limiter := NewTokenBucket(rate, burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			logger.Warn("IP被限流",
				zap.String("ip", ip),
				zap.String("path", c.Request.URL.Path),
			)

			c.JSON(http.StatusTooManyRequests, model.Response{
				Code:    http.StatusTooManyRequests,
				Message: "请求过于频繁,请稍后再试",
				Data:    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"huoxing-search/internal/model"
	"huoxing-search/internal/pkg/config"
	"huoxing-search/internal/pkg/database"
	"huoxing-search/internal/pkg/redis"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	cfg       *config.Config
	startTime time.Time
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(cfg *config.Config) *HealthHandler {
	return &HealthHandler{
		cfg:       cfg,
		startTime: time.Now(),
	}
}

// Health 健康检查接口
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(gin.H{
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
		"version": "1.0.0",
	}))
}

// Ping 简单ping接口
func (h *HealthHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// Ready 就绪检查 (检查依赖服务是否可用)
func (h *HealthHandler) Ready(c *gin.Context) {
	checks := make(map[string]string)

	// 检查MySQL连接
	db := database.GetDB()
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Ping(); err == nil {
			checks["mysql"] = "ok"
		} else {
			checks["mysql"] = "error: " + err.Error()
		}
	} else {
		checks["mysql"] = "error: " + err.Error()
	}

	// 检查Redis连接
	rdb := redis.GetClient()
	if err := rdb.Ping(c.Request.Context()).Err(); err == nil {
		checks["redis"] = "ok"
	} else {
		checks["redis"] = "error: " + err.Error()
	}

	// 判断整体状态
	allOK := true
	for _, status := range checks {
		if status != "ok" {
			allOK = false
			break
		}
	}

	if allOK {
		c.JSON(http.StatusOK, model.Success(checks))
	} else {
		c.JSON(http.StatusServiceUnavailable, model.Response{
			Code:    http.StatusServiceUnavailable,
			Message: "service not ready",
			Data:    checks,
		})
	}
}

// Metrics 系统指标接口
func (h *HealthHandler) Metrics(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 计算运行时长
	uptime := time.Since(h.startTime)

	metrics := gin.H{
		"uptime": gin.H{
			"seconds": int(uptime.Seconds()),
			"human":   uptime.String(),
		},
		"memory": gin.H{
			"alloc_mb":       m.Alloc / 1024 / 1024,
			"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
			"sys_mb":         m.Sys / 1024 / 1024,
			"heap_alloc_mb":  m.HeapAlloc / 1024 / 1024,
			"heap_sys_mb":    m.HeapSys / 1024 / 1024,
			"heap_idle_mb":   m.HeapIdle / 1024 / 1024,
			"heap_inuse_mb":  m.HeapInuse / 1024 / 1024,
			"gc_count":       m.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
		"cpu_cores":  runtime.NumCPU(),
		"go_version": runtime.Version(),
	}

	// 获取数据库连接池状态
	db := database.GetDB()
	if sqlDB, err := db.DB(); err == nil {
		stats := sqlDB.Stats()
		metrics["database"] = gin.H{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"wait_count":       stats.WaitCount,
			"wait_duration_ms": stats.WaitDuration.Milliseconds(),
			"max_idle_closed":  stats.MaxIdleClosed,
			"max_open":         h.cfg.Database.MaxOpenConns,
		}
	}

	c.JSON(http.StatusOK, model.Success(metrics))
}

// Version 版本信息接口
func (h *HealthHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(gin.H{
		"version":    "1.0.0",
		"build_time": "2025-01-14",
		"go_version": runtime.Version(),
		"git_commit": "unknown", // TODO: 从构建参数注入
	}))
}
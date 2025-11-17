package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StatsHandler 统计数据处理器
type StatsHandler struct {
	db *gorm.DB
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(db *gorm.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

// StatsResponse 统计数据响应
type StatsResponse struct {
	TotalSources   int64 `json:"total_sources"`   // 总资源数
	TodaySources   int64 `json:"today_sources"`   // 今日新增资源
	TotalUsers     int64 `json:"total_users"`     // 总用户数
	TodaySearches  int64 `json:"today_searches"`  // 今日搜索次数（暂时返回0）
	TotalCategories int64 `json:"total_categories"` // 总分类数
	TotalAPIs      int64 `json:"total_apis"`      // 搜索线路数
}

// GetDashboardStats 获取仪表盘统计数据
func (h *StatsHandler) GetDashboardStats(c *gin.Context) {
	var stats StatsResponse

	// 1. 统计总资源数
	h.db.Table("qf_source").Where("status = ?", 1).Count(&stats.TotalSources)

	// 2. 统计今日新增资源
	todayStart := time.Now().Truncate(24 * time.Hour).Unix()
	h.db.Table("qf_source").
		Where("status = ? AND create_time >= ?", 1, todayStart).
		Count(&stats.TodaySources)

	// 3. 统计总用户数
	h.db.Table("qf_admin").Where("status = ?", 1).Count(&stats.TotalUsers)

	// 4. 统计总分类数
	h.db.Table("qf_source_category").Where("status = ?", 1).Count(&stats.TotalCategories)

	// 5. 统计搜索线路数
	h.db.Table("qf_api_list").Where("status = ?", 1).Count(&stats.TotalAPIs)

	// 6. 今日搜索次数（暂无搜索日志表，返回0）
	stats.TodaySearches = 0

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    stats,
	})
}

// GetResourceStats 获取资源统计（按分类）
func (h *StatsHandler) GetResourceStats(c *gin.Context) {
	type CategoryStat struct {
		CategoryName string `json:"category_name"`
		Count        int64  `json:"count"`
	}

	var stats []CategoryStat

	h.db.Table("qf_source s").
		Select("c.cate_name as category_name, COUNT(s.source_id) as count").
		Joins("LEFT JOIN qf_source_category c ON s.cate_id = c.cate_id").
		Where("s.status = ?", 1).
		Group("s.cate_id, c.cate_name").
		Order("count DESC").
		Limit(10).
		Scan(&stats)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    stats,
	})
}

// GetRecentSources 获取最近添加的资源
func (h *StatsHandler) GetRecentSources(c *gin.Context) {
	type RecentSource struct {
		SourceID   int64  `json:"source_id"`
		Title      string `json:"title"`
		IsType     int    `json:"is_type"`
		CreateTime int64  `json:"create_time"`
	}

	var sources []RecentSource

	h.db.Table("qf_source").
		Select("source_id, title, is_type, create_time").
		Where("status = ?", 1).
		Order("create_time DESC").
		Limit(10).
		Scan(&sources)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    sources,
	})
}
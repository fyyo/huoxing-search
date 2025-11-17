package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// FrontendHandler 前端页面处理器
type FrontendHandler struct{}

// NewFrontendHandler 创建前端处理器
func NewFrontendHandler() *FrontendHandler {
	return &FrontendHandler{}
}

// RegisterRoutes 注册前端路由
func (h *FrontendHandler) RegisterRoutes(r *gin.Engine) {
	// 前台页面路由
	index := r.Group("/")
	{
		index.GET("", h.Home)
		index.GET("/search", h.Search)
		index.GET("/detail/:id", h.Detail)
	}

	// 后台页面路由
	admin := r.Group("/admin")
	{
		admin.GET("/login", h.AdminLogin)
		
		// 需要认证的后台页面
		authAdmin := admin.Group("")
		// TODO: 添加认证中间件
		// authAdmin.Use(middleware.AdminAuth())
		{
			authAdmin.GET("", h.AdminDashboard)
			authAdmin.GET("/dashboard", h.AdminDashboard)
			
			// 资源管理
			authAdmin.GET("/source/list", h.AdminSourceList)
			authAdmin.GET("/source/category", h.AdminSourceCategory)
			authAdmin.GET("/source/import", h.AdminSourceImport)
			
			// 搜索配置
			authAdmin.GET("/search/api", h.AdminSearchAPI)
			
			// 用户管理
			authAdmin.GET("/user", h.AdminUser)
			
			// 系统设置
			authAdmin.GET("/system/config", h.AdminSystemConfig)
			authAdmin.GET("/system/netdisk", h.AdminSystemNetdisk)
			authAdmin.GET("/system/wechat", h.AdminSystemWechat)
		}
	}
}

// Home 首页
func (h *FrontendHandler) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index/home_simple.html", gin.H{
		"Title": "心悦网盘搜索",
	})
}

// Search 搜索页面
func (h *FrontendHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	panType := c.DefaultQuery("pan_type", "0")
	
	c.HTML(http.StatusOK, "index/search_simple.html", gin.H{
		"Title":   keyword + " - 搜索结果",
		"Keyword": keyword,
		"PanType": panType,
	})
}

// Detail 详情页面
func (h *FrontendHandler) Detail(c *gin.Context) {
	id := c.Param("id")
	
	// TODO: 从数据库获取资源详情
	
	c.HTML(http.StatusOK, "index/detail.html", gin.H{
		"Title": "资源详情",
		"ID":    id,
	})
}

// AdminLogin 后台登录页
func (h *FrontendHandler) AdminLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/login.html", gin.H{
		"Title": "管理员登录",
	})
}

// AdminDashboard 后台首页
func (h *FrontendHandler) AdminDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/dashboard.html", gin.H{
		"Title":       "控制台",
		"Username":    "admin",
		"ActiveMenu":  "/admin",
		"Breadcrumbs": []string{},
	})
}

// AdminSourceList 资源列表
func (h *FrontendHandler) AdminSourceList(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/sources_simple.html", gin.H{
		"Title":       "资源列表",
		"Username":    "admin",
		"ActiveMenu":  "/admin/source/list",
		"Breadcrumbs": []string{"资源管理", "资源列表"},
	})
}

// AdminSourceCategory 资源分类
func (h *FrontendHandler) AdminSourceCategory(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/category.html", gin.H{
		"Title":       "分类管理",
		"Username":    "admin",
		"ActiveMenu":  "/admin/source/category",
		"Breadcrumbs": []string{"资源管理", "分类管理"},
	})
}


// AdminSourceImport 批量导入
func (h *FrontendHandler) AdminSourceImport(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/batch_import.html", gin.H{
		"Title":       "批量导入",
		"Username":    "admin",
		"ActiveMenu":  "/admin/source/import",
		"Breadcrumbs": []string{"资源管理", "批量导入"},
	})
}

// AdminSearchAPI 搜索线路
func (h *FrontendHandler) AdminSearchAPI(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/api_config.html", gin.H{
		"Title":       "搜索线路配置",
		"Username":    "admin",
		"ActiveMenu":  "/admin/search/api",
		"Breadcrumbs": []string{"搜索配置", "搜索线路"},
	})
}

// AdminUser 用户管理
func (h *FrontendHandler) AdminUser(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/admin_management.html", gin.H{
		"Title":       "管理员管理",
		"Username":    "admin",
		"ActiveMenu":  "/admin/user",
		"Breadcrumbs": []string{"用户管理"},
	})
}

// AdminSystemConfig 基本配置
func (h *FrontendHandler) AdminSystemConfig(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/system_config.html", gin.H{
		"Title":       "系统配置",
		"Username":    "admin",
		"ActiveMenu":  "/admin/system/config",
		"Breadcrumbs": []string{"系统设置", "基本配置"},
	})
}

// AdminSystemNetdisk 网盘配置
func (h *FrontendHandler) AdminSystemNetdisk(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/netdisk_config.html", gin.H{
		"Title":       "网盘配置",
		"Username":    "admin",
		"ActiveMenu":  "/admin/system/netdisk",
		"Breadcrumbs": []string{"系统设置", "网盘配置"},
	})
}


// AdminSystemWechat 微信配置
func (h *FrontendHandler) AdminSystemWechat(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/wechat_config.html", gin.H{
		"Title":       "微信配置",
		"Username":    "admin",
		"ActiveMenu":  "/admin/system/wechat",
		"Breadcrumbs": []string{"系统设置", "微信配置"},
	})
}

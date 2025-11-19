package api

import (
	"html/template"
	"path/filepath"
	
	"github.com/gin-gonic/gin"
	"huoxing-search/internal/middleware"
	"huoxing-search/internal/pkg/config"
	"huoxing-search/internal/pkg/database"
	"huoxing-search/internal/repository"
	"huoxing-search/internal/service"
)

// SetupRouter 设置路由
func SetupRouter(cfg *config.Config) *gin.Engine {
	// 设置运行模式
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// 全局中间件
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimitMiddleware(cfg))
	
	// 注册安装路由（最高优先级，不需要其他依赖）
	installHandler := NewInstallHandler()
	installHandler.RegisterInstallRoutes(r)

	// 设置模板函数
	r.SetFuncMap(template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
	})
	
	// 加载HTML模板 - 使用glob模式，Windows和Linux都兼容
	r.LoadHTMLGlob(filepath.Join("web", "templates", "*", "*.html"))
	
	// 静态文件服务
	r.Static("/static", "./web/static")
	
	// 前端页面路由
	frontendHandler := NewFrontendHandler()
	frontendHandler.RegisterRoutes(r)

	// API分组
	api := r.Group("/api")
	{
		// 公开接口
		public := api.Group("")
		{
			// 健康检查接口
			healthHandler := NewHealthHandler(cfg)
			public.GET("/health", healthHandler.Health)
			public.GET("/ping", healthHandler.Ping)
			public.GET("/ready", healthHandler.Ready)
			public.GET("/metrics", healthHandler.Metrics)
			public.GET("/version", healthHandler.Version)

			// 认证接口
			authHandler := NewAuthHandler(cfg)
			public.POST("/auth/login", authHandler.Login)
			public.POST("/auth/register", authHandler.Register)
			public.POST("/auth/refresh", authHandler.RefreshToken)
			
			// 管理员登录接口（为了兼容前端）
			public.POST("/admin/login", authHandler.Login)

			// 初始化仓储和服务
			configRepo := repository.NewConfigRepository()
			cacheRepo := repository.NewCacheRepository()
			
			// 转存服务（需要先创建，因为搜索服务依赖它）
			transferService := service.NewTransferService(cfg)
			transferHandler := NewTransferHandler(cfg)
			public.POST("/transfer", transferHandler.Transfer)
			public.POST("/transfer/save", transferHandler.TransferAndSave)
			
			// 搜索接口（传入转存服务）
			searchService := service.NewSearchService(configRepo, cacheRepo, transferService)
			searchHandler := NewSearchHandler(searchService)
			public.POST("/search", searchHandler.Search)
			public.DELETE("/search/cache", searchHandler.ClearCache)

			// 微信回调接口（无需认证）
			wechatHandler := NewWechatHandler(configRepo)
			
			// 微信对话开放平台回调
			// 回调地址：https://您的域名/api/wechat/chatbot/callback
			public.POST("/wechat/chatbot/callback", wechatHandler.ChatbotCallback)
			public.GET("/wechat/chatbot/callback", wechatHandler.ChatbotCallback)
			
			// 微信公众号回调
			// 验证地址（GET）：https://您的域名/api/wechat/official/callback
			public.GET("/wechat/official/callback", wechatHandler.OfficialAccountVerify)
			// 消息回调（POST）：https://您的域名/api/wechat/official/callback
			public.POST("/wechat/official/callback", wechatHandler.OfficialAccountCallback)
		}

		// 需要认证的接口
		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware(cfg))
		{
			// 用户相关接口
			authHandler2 := NewAuthHandler(cfg)
			auth.GET("/auth/userinfo", authHandler2.GetUserInfo)
			auth.POST("/auth/logout", authHandler2.Logout)

			// 资源管理
			sourceHandler := NewSourceHandler(cfg)
			auth.GET("/sources", sourceHandler.List)
			auth.GET("/sources/:id", sourceHandler.GetByID)
			auth.POST("/sources", sourceHandler.Create)
			auth.PUT("/sources", sourceHandler.Update)      // 接受body中的source_id
			auth.DELETE("/sources", sourceHandler.Delete)   // 接受body中的ids数组

			// 用户管理(仅管理员)
			admin := auth.Group("/admin")
			admin.Use(middleware.AdminMiddleware())
			{
				// 用户管理
				userHandler := NewUserHandler(cfg)
				admin.GET("/users", userHandler.List)
				admin.GET("/users/:id", userHandler.GetByID)
				admin.POST("/users/create", userHandler.Create)
				admin.POST("/users/update", userHandler.Update)
				admin.POST("/users/delete", userHandler.Delete)
				admin.POST("/users/reset-password", userHandler.ResetPassword)

				// 资源管理
				admin.POST("/sources/create", sourceHandler.Create)
				admin.POST("/sources/update", sourceHandler.Update)
				admin.POST("/sources/delete", sourceHandler.Delete)

				// API配置管理
				apiConfigHandler := NewApiConfigHandler()
				admin.GET("/apis", apiConfigHandler.List)
				admin.GET("/apis/:id", apiConfigHandler.GetByID)
				admin.POST("/apis/create", apiConfigHandler.Create)
				admin.POST("/apis/update", apiConfigHandler.Update)
				admin.POST("/apis/delete", apiConfigHandler.Delete)
				admin.POST("/apis/status", apiConfigHandler.UpdateStatus)

				// 配置测试
				configTestHandler := NewConfigTestHandler(cfg)
				admin.POST("/test/netdisk", configTestHandler.TestNetdiskConnection)
				admin.POST("/test/wechat", configTestHandler.TestWechatConnection)

				// 系统配置管理
				systemConfigHandler := NewSystemConfigHandler()
				admin.GET("/configs", systemConfigHandler.List)
				admin.GET("/configs/:id", systemConfigHandler.GetByID)
				admin.GET("/configs/name/:name", systemConfigHandler.GetByName)
				admin.POST("/configs/create", systemConfigHandler.Create)
				admin.POST("/configs/update", systemConfigHandler.Update)
				admin.POST("/configs/delete", systemConfigHandler.Delete)
				admin.PUT("/configs/batch", systemConfigHandler.BatchUpdate)  // 批量更新（需要conf_id）
				admin.POST("/configs/batch-upsert", systemConfigHandler.BatchUpsert)  // 批量插入或更新（根据name）

				// 管理员管理
				adminManagementHandler := NewAdminManagementHandler()
				admin.GET("/admins", adminManagementHandler.List)
				admin.GET("/admins/:id", adminManagementHandler.GetByID)
				admin.POST("/admins/create", adminManagementHandler.Create)
				admin.POST("/admins/update", adminManagementHandler.Update)
				admin.POST("/admins/delete", adminManagementHandler.Delete)
				admin.POST("/admins/reset-password", adminManagementHandler.ResetPassword)
				admin.POST("/admins/status", adminManagementHandler.UpdateStatus)

				// 分类管理
				categoryHandler := NewCategoryHandler()
				admin.GET("/categories", categoryHandler.List)
				admin.GET("/categories/:id", categoryHandler.GetByID)
				admin.POST("/categories/create", categoryHandler.Create)
				admin.POST("/categories/update", categoryHandler.Update)
				admin.POST("/categories/delete", categoryHandler.Delete)

				// 批量导入
				batchImportHandler := NewBatchImportHandler()
				admin.POST("/batch-import", batchImportHandler.Import)
				admin.GET("/batch-import/template", batchImportHandler.GetTemplate)

				// 统计数据
				statsHandler := NewStatsHandler(database.GetDB())
				admin.GET("/stats/dashboard", statsHandler.GetDashboardStats)
				admin.GET("/stats/resources", statsHandler.GetResourceStats)
				admin.GET("/stats/recent", statsHandler.GetRecentSources)
			}
		}
	}

	return r
}
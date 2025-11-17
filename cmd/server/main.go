package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"xinyue-go/internal/api"
	"xinyue-go/internal/netdisk"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/pkg/database"
	"xinyue-go/internal/pkg/logger"
	"xinyue-go/internal/pkg/redis"
	"xinyue-go/internal/repository"
	"xinyue-go/internal/service"
	"xinyue-go/pansou"
)

var (
	// 全局路由器和服务器，用于动态切换
	globalRouter  *gin.Engine
	globalServer  *http.Server
	globalConfig  *config.Config
	routerMutex   sync.RWMutex
	isInstallMode bool
)

func main() {
	// 检查是否已安装
	installLockPath := "./install.lock"
	configPath := "./config.yaml"
	
	// 如果没有安装锁文件或配置文件，进入安装模式
	if !fileExists(installLockPath) || !fileExists(configPath) {
		fmt.Println("系统未安装，启动安装向导...")
		isInstallMode = true
		runUnifiedMode(true)
		return
	}

	// 正常模式启动
	isInstallMode = false
	runUnifiedMode(false)
}

// runUnifiedMode 统一的运行模式（支持动态切换）
func runUnifiedMode(installMode bool) {
	var cfg *config.Config
	var err error

	if installMode {
		// 安装模式：使用简单配置
		gin.SetMode(gin.ReleaseMode)
		fmt.Println("\n===========================================")
		fmt.Println("  Xinyue-Go 安装向导已启动")
		fmt.Println("  请在浏览器中访问: http://localhost:6060/install")
		fmt.Println("===========================================\n")
	} else {
		// 正常模式：加载完整配置
		cfg, err = config.LoadConfig("./config.yaml")
		if err != nil {
			fmt.Printf("加载配置失败: %v\n", err)
			os.Exit(1)
		}

		// 初始化日志
		if err := logger.InitLogger(
			cfg.Log.Level,
			cfg.Log.FilePath,
			cfg.Log.MaxSize,
			cfg.Log.MaxBackups,
			cfg.Log.MaxAge,
		); err != nil {
			fmt.Printf("初始化日志失败: %v\n", err)
			os.Exit(1)
		}
		defer logger.Sync()

		logger.Info("Xinyue-Go 启动中...")

		// 初始化数据库
		if err := database.InitMySQL(&cfg.Database); err != nil {
			logger.Fatal("初始化MySQL失败", zap.Error(err))
		}
		defer database.Close()
		logger.Info("MySQL连接成功")

		// 初始化Redis（可选）
		if err := redis.InitRedis(&cfg.Redis); err != nil {
			logger.Warn("Redis连接失败，缓存功能将不可用", zap.Error(err))
		} else {
			defer redis.Close()
			logger.Info("Redis连接成功")
		}
		
		// 初始化Pansou搜索引擎
		if err := pansou.Init(); err != nil {
			logger.Fatal("初始化Pansou搜索引擎失败", zap.Error(err))
		}
		logger.Info("Pansou搜索引擎初始化成功")
		
		// 启动临时资源清理任务（每天凌晨3点执行）
		go startCleanupTask(cfg)
	}

	// 保存全局配置
	globalConfig = cfg

	// 创建路由
	router := createRouter(installMode, cfg)
	globalRouter = router

	// 创建HTTP服务器
	port := 6060
	if !installMode && cfg != nil {
		port = cfg.Server.Port
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(dynamicHandler),
	}
	globalServer = server

	if !installMode {
		server.ReadTimeout = time.Duration(cfg.Server.ReadTimeout) * time.Second
		server.WriteTimeout = time.Duration(cfg.Server.WriteTimeout) * time.Second
		server.MaxHeaderBytes = 1 << 20
	}

	// 启动服务器
	go func() {
		if !installMode {
			logger.Info("服务器启动",
				zap.Int("port", port),
				zap.String("mode", cfg.Server.Mode),
			)
		}
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if installMode {
				fmt.Printf("服务器启动失败: %v\n", err)
			} else {
				logger.Fatal("服务器启动失败", zap.Error(err))
			}
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if installMode {
		fmt.Println("正在关闭服务器...")
	} else {
		logger.Info("正在关闭服务器...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		if installMode {
			fmt.Printf("服务器关闭异常: %v\n", err)
		} else {
			logger.Error("服务器关闭异常", zap.Error(err))
		}
	}

	if installMode {
		fmt.Println("服务器已关闭")
	} else {
		logger.Info("服务器已关闭")
	}
}

// dynamicHandler 动态处理器，根据当前模式转发请求
func dynamicHandler(w http.ResponseWriter, r *http.Request) {
	routerMutex.RLock()
	router := globalRouter
	routerMutex.RUnlock()

	if router != nil {
		router.ServeHTTP(w, r)
	} else {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}
}

// createRouter 创建路由
func createRouter(installMode bool, cfg *config.Config) *gin.Engine {
	if installMode {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg != nil {
		gin.SetMode(cfg.Server.Mode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	if installMode {
		// 安装模式：只注册安装路由
		installHandler := api.NewInstallHandler()
		
		// 设置安装完成回调
		installHandler.SetReloadCallback(func() error {
			return switchToNormalMode()
		})
		
		installHandler.RegisterInstallRoutes(router)

		// 根路径重定向到安装页面
		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/install")
		})
	} else {
		// 正常模式：注册所有路由
		router = api.SetupRouter(cfg)
	}

	return router
}

// switchToNormalMode 从安装模式切换到正常模式
func switchToNormalMode() error {
	fmt.Println("\n===========================================")
	fmt.Println("  安装完成，正在切换到正常模式...")
	fmt.Println("===========================================\n")

	// 等待一下，让安装请求完成
	time.Sleep(500 * time.Millisecond)

	// 加载配置
	cfg, err := config.LoadConfig("./config.yaml")
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// 初始化日志
	if err := logger.InitLogger(
		cfg.Log.Level,
		cfg.Log.FilePath,
		cfg.Log.MaxSize,
		cfg.Log.MaxBackups,
		cfg.Log.MaxAge,
	); err != nil {
		return fmt.Errorf("初始化日志失败: %v", err)
	}

	logger.Info("安装完成，正在初始化系统...")

	// 初始化数据库
	if err := database.InitMySQL(&cfg.Database); err != nil {
		return fmt.Errorf("初始化MySQL失败: %v", err)
	}
	logger.Info("MySQL连接成功")

	// 初始化Redis（可选）
	if err := redis.InitRedis(&cfg.Redis); err != nil {
		logger.Warn("Redis连接失败，缓存功能将不可用", zap.Error(err))
	} else {
		logger.Info("Redis连接成功")
	}

	// 初始化Pansou搜索引擎
	if err := pansou.Init(); err != nil {
		return fmt.Errorf("初始化Pansou搜索引擎失败: %v", err)
	}
	logger.Info("Pansou搜索引擎初始化成功")

	// 创建新路由
	newRouter := api.SetupRouter(cfg)

	// 原子性地切换路由
	routerMutex.Lock()
	globalRouter = newRouter
	globalConfig = cfg
	isInstallMode = false
	routerMutex.Unlock()

	logger.Info("系统已切换到正常模式，所有功能已就绪")
	fmt.Println("\n===========================================")
	fmt.Println("  ✅ 系统初始化完成！")
	fmt.Println("  现在可以访问管理后台: http://localhost:6060/admin/login")
	fmt.Println("===========================================\n")

	return nil
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// startCleanupTask 启动临时资源清理任务
func startCleanupTask(cfg *config.Config) {
	// 创建依赖
	configRepo := repository.NewConfigRepository()
	netdiskManager := netdisk.NewNetdiskManager(cfg)
	
	// 创建清理服务
	cleanupService := service.NewCleanupService(configRepo, netdiskManager)
	ctx := context.Background()
	
	// 每24小时执行一次清理
	cleanupService.StartScheduledCleanup(ctx, 24*time.Hour)
}
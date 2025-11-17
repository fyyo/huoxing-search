package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

// InstallHandler 安装处理器
type InstallHandler struct {
	installLockFile string
	sqlFile         string
	reloadCallback  func() error // 安装完成后的重载回调
}

// NewInstallHandler 创建安装处理器
func NewInstallHandler() *InstallHandler {
	return &InstallHandler{
		installLockFile: "./install.lock",
		sqlFile:         "./install/data.sql",
	}
}

// SetReloadCallback 设置重载回调函数
func (h *InstallHandler) SetReloadCallback(callback func() error) {
	h.reloadCallback = callback
}

// InstallRequest 安装请求
type InstallRequest struct {
	DBHost     string `json:"db_host" binding:"required"`
	DBPort     int    `json:"db_port" binding:"required"`
	DBUser     string `json:"db_user" binding:"required"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name" binding:"required"`
	DBPrefix   string `json:"db_prefix" binding:"required"`
	DBSSLMode  bool   `json:"db_ssl_mode"` // SSL验证选项
	SiteName   string `json:"site_name" binding:"required"`
	AdminUser  string `json:"admin_user" binding:"required"`
	AdminPass  string `json:"admin_pass" binding:"required"`
}

// RegisterInstallRoutes 注册安装路由
func (h *InstallHandler) RegisterInstallRoutes(r *gin.Engine) {
	install := r.Group("/install")
	{
		// 检查是否已安装
		install.Use(h.CheckInstalled())
		
		// 安装页面
		install.GET("/", h.ShowInstallPage)
		install.GET("", h.ShowInstallPage)
		// 检查环境
		install.GET("/check", h.CheckEnvironment)
		// 测试数据库连接
		install.POST("/test-db", h.TestDatabase)
		// 执行安装
		install.POST("/execute", h.ExecuteInstall)
	}
}

// CheckInstalled 检查是否已安装的中间件
func (h *InstallHandler) CheckInstalled() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := os.Stat(h.installLockFile); err == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "系统已安装！如需重新安装，请删除 install.lock 文件",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ShowInstallPage 显示安装页面
func (h *InstallHandler) ShowInstallPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, getInstallHTML())
}

// CheckEnvironment 检查运行环境
func (h *InstallHandler) CheckEnvironment(c *gin.Context) {
	checks := []map[string]interface{}{
		{
			"name":   "Go 版本",
			"status": true,
			"value":  "1.21+",
		},
		{
			"name":   "配置目录可写",
			"status": h.checkDirWritable("./"),
			"value":  "./",
		},
		{
			"name":   "数据文件存在",
			"status": h.checkFileExists(h.sqlFile),
			"value":  h.sqlFile,
		},
	}

	allPass := true
	for _, check := range checks {
		if !check["status"].(bool) {
			allPass = false
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     200,
		"message":  "环境检查完成",
		"all_pass": allPass,
		"checks":   checks,
	})
}

// TestDatabase 测试数据库连接
func (h *InstallHandler) TestDatabase(c *gin.Context) {
	var req InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 测试连接
	tlsParam := ""
	if req.DBSSLMode {
		tlsParam = "&tls=skip-verify"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4%s",
		req.DBUser, req.DBPassword, req.DBHost, req.DBPort, tlsParam)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "数据库连接失败: " + err.Error(),
		})
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "数据库连接失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "数据库连接成功",
	})
}

// ExecuteInstall 执行安装
func (h *InstallHandler) ExecuteInstall(c *gin.Context) {
	var req InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 1. 连接数据库
	tlsParam := ""
	if req.DBSSLMode {
		tlsParam = "&tls=skip-verify"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4%s",
		req.DBUser, req.DBPassword, req.DBHost, req.DBPort, tlsParam)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "数据库连接失败: " + err.Error(),
		})
		return
	}
	defer db.Close()

	// 2. 创建数据库
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", req.DBName))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "创建数据库失败: " + err.Error(),
		})
		return
	}

	// 3. 选择数据库
	_, err = db.Exec(fmt.Sprintf("USE `%s`", req.DBName))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "选择数据库失败: " + err.Error(),
		})
		return
	}

	// 4. 读取SQL文件并执行
	sqlContent, err := os.ReadFile(h.sqlFile)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "读取SQL文件失败: " + err.Error(),
		})
		return
	}

	// 替换表前缀
	sqlStr := string(sqlContent)
	sqlStr = strings.ReplaceAll(sqlStr, "qf_", req.DBPrefix)

	// 分割并执行SQL语句
	statements := strings.Split(sqlStr, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    500,
				"message": "执行SQL失败: " + err.Error(),
				"sql":     stmt[:min(len(stmt), 100)],
			})
			return
		}
	}

	// 5. 创建管理员账户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.AdminPass), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "密码加密失败: " + err.Error(),
		})
		return
	}

	insertAdmin := fmt.Sprintf(`
		INSERT INTO %sadmin (username, password, status, create_time, update_time)
		VALUES (?, ?, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP())
	`, req.DBPrefix)
	
	_, err = db.Exec(insertAdmin, req.AdminUser, string(hashedPassword))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "创建管理员失败: " + err.Error(),
		})
		return
	}

	// 6. 生成配置文件
	configContent := fmt.Sprintf(`server:
  port: 6060
  mode: release
  read_timeout: 60
  write_timeout: 60

database:
  host: %s
  port: %d
  username: %s
  password: %s
  database: %s
  prefix: %s
  max_idle_conns: 10
  max_open_conns: 100
  ssl_mode: %t

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

pansou:
  url: http://localhost:8888

jwt:
  secret: %s
  expire_hours: 168

log:
  level: info
  file_path: ./logs/app.log
  max_size: 100
  max_backups: 3
  max_age: 7

app:
  name: %s
  debug: false
`, req.DBHost, req.DBPort, req.DBUser, req.DBPassword, req.DBName, req.DBPrefix,
		req.DBSSLMode, h.generateRandomString(32), req.SiteName)

	if err := os.WriteFile("./config.yaml", []byte(configContent), 0644); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "生成配置文件失败: " + err.Error(),
		})
		return
	}

	// 7. 创建安装锁文件
	if err := os.WriteFile(h.installLockFile, []byte("installed"), 0644); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "创建锁文件失败: " + err.Error(),
		})
		return
	}

	// 8. 触发系统重载（如果设置了回调）
	if h.reloadCallback != nil {
		go func() {
			// 延迟1秒后重载，确保响应已发送
			// time.Sleep(1 * time.Second)
			h.reloadCallback()
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     200,
		"message":  "安装成功！系统正在自动初始化...",
		"redirect": "/admin/login",
	})
}

// 辅助函数
func (h *InstallHandler) checkDirWritable(dir string) bool {
	testFile := filepath.Join(dir, ".write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return false
	}
	os.Remove(testFile)
	return true
}

func (h *InstallHandler) checkFileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func (h *InstallHandler) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
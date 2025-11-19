package database

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"huoxing-search/internal/pkg/config"
)

var DB *gorm.DB

// InitMySQL 初始化MySQL连接
func InitMySQL(cfg *config.DatabaseConfig) error {
	// 如果启用了SSL，注册TLS配置
	if cfg.SSLMode {
		if err := registerTLSConfig(cfg); err != nil {
			return fmt.Errorf("注册TLS配置失败: %w", err)
		}
	}

	// 设置日志级别
	var logLevel logger.LogLevel
	switch cfg.LogLevel {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Info
	}

	// 连接数据库
	db, err := gorm.Open(gorm_mysql.Open(cfg.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return fmt.Errorf("连接MySQL失败: %w", err)
	}

	// 获取底层的sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取sql.DB失败: %w", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping MySQL失败: %w", err)
	}

	DB = db
	return nil
}

// registerTLSConfig 注册TLS配置
func registerTLSConfig(cfg *config.DatabaseConfig) error {
	// 如果没有提供证书文件，使用简化的TLS模式（如TiDB Cloud）
	if cfg.SSLRootCert == "" && cfg.SSLCert == "" && cfg.SSLKey == "" {
		// 不需要注册自定义TLS配置
		return nil
	}

	tlsConfigName := "custom"
	if cfg.TLSConfig != "" {
		tlsConfigName = cfg.TLSConfig
	}

	// 创建TLS配置
	tlsConfig := &tls.Config{}

	// 加载CA根证书
	if cfg.SSLRootCert != "" {
		caCert, err := os.ReadFile(cfg.SSLRootCert)
		if err != nil {
			return fmt.Errorf("读取CA根证书失败: %w", err)
		}
		
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("解析CA根证书失败")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// 加载客户端证书和密钥（可选）
	if cfg.SSLCert != "" && cfg.SSLKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.SSLCert, cfg.SSLKey)
		if err != nil {
			return fmt.Errorf("加载客户端证书失败: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 对于云数据库，通常需要验证服务器名称
	// 但某些情况下可能需要跳过验证（不推荐生产环境）
	// tlsConfig.InsecureSkipVerify = true

	// 注册TLS配置
	if err := mysql.RegisterTLSConfig(tlsConfigName, tlsConfig); err != nil {
		return fmt.Errorf("注册TLS配置失败: %w", err)
	}

	return nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return DB
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
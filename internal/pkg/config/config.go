package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Pansou    PansouConfig    `mapstructure:"pansou"`
	Cache     CacheConfig     `mapstructure:"cache"`
	Transfer  TransferConfig  `mapstructure:"transfer"`
	Netdisk   NetdiskConfig   `mapstructure:"netdisk"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Log       LogConfig       `mapstructure:"log"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type ServerConfig struct {
	Port         int `mapstructure:"port"`
	Mode         string
	ReadTimeout  int `mapstructure:"read_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host         string
	Port         int
	Username     string
	Password     string
	Database     string
	Prefix       string `mapstructure:"prefix"`         // 表前缀
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	LogLevel     string `mapstructure:"log_level"`
	
	// SSL配置 (用于云数据库如 TiDB Cloud)
	SSLMode      bool   `mapstructure:"ssl_mode"`       // 是否启用SSL
	SSLCert      string `mapstructure:"ssl_cert"`       // 客户端证书路径
	SSLKey       string `mapstructure:"ssl_key"`        // 客户端密钥路径
	SSLRootCert  string `mapstructure:"ssl_root_cert"`  // CA根证书路径
	TLSConfig    string `mapstructure:"tls_config"`     // TLS配置名称 (如 "tidb")
}

type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int `mapstructure:"pool_size"`
	MinIdleConns int `mapstructure:"min_idle_conns"`
}

type PansouConfig struct {
	URL     string
	Timeout int
}

type CacheConfig struct {
	SearchTTL int `mapstructure:"search_ttl"`
}

type TransferConfig struct {
	MaxConcurrent int `mapstructure:"max_concurrent"`
	Timeout       int
	MaxSuccess    int `mapstructure:"max_success"`
}

type NetdiskConfig struct {
	Quark  NetdiskAccount
	Baidu  NetdiskAccount
	Aliyun NetdiskAccount
	UC     NetdiskAccount
	Xunlei NetdiskAccount
}

type NetdiskAccount struct {
	Cookie       string
	AccessToken  string `mapstructure:"access_token"`
	RefreshToken string `mapstructure:"refresh_token"`
	Token        string
}

type JWTConfig struct {
	Secret      string
	Expiration  int `mapstructure:"expiration"`  // 过期时间（小时）
	ExpireHours int `mapstructure:"expire_hours"` // 兼容旧配置
}

type LogConfig struct {
	Level      string
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

type RateLimitConfig struct {
	Enabled            bool
	Rate               float64 `mapstructure:"rate"` // 兼容rate字段
	RequestsPerSecond  float64 `mapstructure:"requests_per_second"`
	Burst              int
}

var GlobalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	GlobalConfig = &config
	return &config, nil
}

func setDefaults() {
	viper.SetDefault("server.port", 6060)
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("server.read_timeout", 60)
	viper.SetDefault("server.write_timeout", 60)
	viper.SetDefault("cache.search_ttl", 60)
	viper.SetDefault("transfer.max_concurrent", 5)
	viper.SetDefault("transfer.timeout", 15)
	viper.SetDefault("transfer.max_success", 2)
	viper.SetDefault("jwt.expire_hours", 24)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("rate_limit.enabled", true)
	viper.SetDefault("rate_limit.requests_per_second", 10)
	viper.SetDefault("rate_limit.burst", 20)
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
	
	// 如果启用了SSL
	if c.SSLMode {
		// 如果没有提供证书文件，使用简化的 tls=true（适用于 TiDB Cloud 等）
		if c.SSLRootCert == "" && c.SSLCert == "" && c.SSLKey == "" {
			dsn += "&tls=skip-verify"
		} else if c.TLSConfig != "" {
			// 使用预配置的TLS配置名称
			dsn += "&tls=" + c.TLSConfig
		} else {
			// 使用自定义TLS（需要在连接前注册）
			dsn += "&tls=custom"
		}
	}
	
	return dsn
}

// GetRedisAddr 获取Redis地址
func (c *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetPansouURL 获取Pansou完整URL
func (c *PansouConfig) GetURL() string {
	return c.URL
}

// GetTimeout 获取超时时间
func (c *PansouConfig) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// GetTransferTimeout 获取转存超时时间
func (c *TransferConfig) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// GetSearchCacheTTL 获取搜索缓存过期时间
func (c *CacheConfig) GetSearchCacheTTL() time.Duration {
	return time.Duration(c.SearchTTL) * time.Second
}
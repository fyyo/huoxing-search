package model

import (
	"time"

	"gorm.io/gorm"
)

// Config 系统配置模型
type Config struct {
	ConfID      int    `gorm:"primaryKey;column:conf_id;autoIncrement" json:"conf_id"`
	Name        string `gorm:"column:name;type:varchar(100);not null;uniqueIndex:uk_name" json:"name"`
	Value       string `gorm:"column:value;type:text" json:"value"`
	Title       string `gorm:"column:title;type:varchar(100)" json:"title"`
	Description string `gorm:"column:description;type:varchar(255)" json:"description"`
	Group       int    `gorm:"column:group;type:tinyint;default:0" json:"group"`
	Type        int    `gorm:"column:type;type:tinyint;default:1" json:"type"`
	Options     string `gorm:"column:options;type:text" json:"options"`
	Sort        int    `gorm:"column:sort;type:int;default:0" json:"sort"`
	Status      int    `gorm:"column:status;type:tinyint;default:1" json:"status"`
	CreateTime  int64  `gorm:"column:create_time" json:"create_time"`
	UpdateTime  int64  `gorm:"column:update_time" json:"update_time"`
}

// TableName 指定表名
func (Config) TableName() string {
	return "qf_conf"
}

// BeforeCreate GORM钩子 - 创建前
func (c *Config) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	c.CreateTime = now
	c.UpdateTime = now
	return nil
}

// BeforeUpdate GORM钩子 - 更新前
func (c *Config) BeforeUpdate(tx *gorm.DB) error {
	c.UpdateTime = time.Now().Unix()
	return nil
}

// 配置分组常量
const (
	ConfigGroupBasic  = 0 // 基本配置
	ConfigGroupSearch = 1 // 搜索配置
	ConfigGroupPan    = 2 // 网盘配置
	ConfigGroupWechat = 3 // 微信配置
)

// 配置名称常量
const (
	// 基本配置
	ConfSiteName    = "site_name"
	ConfKeywords    = "keywords"
	ConfDescription = "description"
	
	// 搜索配置
	ConfMaxSearchResults = "max_search_results"
	ConfCacheExpire      = "cache_expire"
	ConfBanKeywords      = "ban_keywords"
	ConfPansouURL        = "pansou_url"
	ConfPansouTimeout    = "pansou_timeout"
	
	// 夸克网盘配置
	ConfQuarkCookie   = "quark_cookie"
	ConfQuarkSavePath = "quark_save_path"
	ConfQuarkBanned   = "quark_banned"
	
	// 百度网盘配置
	ConfBaiduCookie      = "baidu_cookie"
	ConfBaiduAccessToken = "baidu_access_token"
	ConfBaiduSavePath    = "baidu_save_path"
	
	// 阿里云盘配置
	ConfAliyunRefreshToken = "aliyun_refresh_token"
	ConfAliyunAccessToken  = "aliyun_access_token"
	ConfAliyunSavePath     = "aliyun_save_path"
	
	// UC网盘配置
	ConfUCCookie   = "uc_cookie"
	ConfUCSavePath = "uc_save_path"
	
	// 迅雷网盘配置
	ConfXunleiToken    = "xunlei_token"
	ConfXunleiSavePath = "xunlei_save_path"
	
	// 微信对话平台配置
	ConfWechatToken  = "wechat_token"
	ConfWechatAesKey = "wechat_aes_key"
	ConfWechatAppID  = "wechat_appid"
	
	// 微信公众号配置
	ConfWechatMpToken  = "wechat_mp_token"
	ConfWechatMpAppID  = "wechat_mp_appid"
	ConfWechatMpSecret = "wechat_mp_secret"
)
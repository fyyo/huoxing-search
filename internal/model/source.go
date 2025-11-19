package model

import (
	"fmt"
	"time"
	"gorm.io/gorm"
)

// Source 资源模型
type Source struct {
	SourceID   uint64 `gorm:"primaryKey;column:source_id;autoIncrement" json:"source_id"`
	Title      string `gorm:"column:title;type:varchar(255);not null" json:"title"`
	URL        string `gorm:"column:url;type:varchar(500);not null" json:"url"`
	Content    string `gorm:"column:content;type:varchar(500)" json:"content,omitempty"`
	IsType     int    `gorm:"column:is_type;type:tinyint;default:0" json:"is_type"` // 0=夸克 2=百度 3=阿里 4=UC 5=迅雷
	Fid        string `gorm:"column:fid;type:varchar(500)" json:"fid,omitempty"`
	IsTime     int    `gorm:"column:is_time;type:tinyint;default:0" json:"is_time"` // 是否临时:0否,1是
	Status     int    `gorm:"column:status;type:tinyint;default:1" json:"status"`
	CreateTime int64  `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime int64  `gorm:"column:update_time;not null" json:"update_time"`
}

// TableName 指定表名
func (Source) TableName() string {
	return "qf_source"
}

// BeforeCreate GORM钩子:创建前
func (s *Source) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	s.CreateTime = now
	s.UpdateTime = now
	return nil
}

// BeforeUpdate GORM钩子:更新前
func (s *Source) BeforeUpdate(tx *gorm.DB) error {
	s.UpdateTime = time.Now().Unix()
	return nil
}

// SearchResult 搜索结果
type SearchResult struct {
	Title         string `json:"title"`
	URL           string `json:"url"`
	Password      string `json:"password,omitempty"`
	Source        string `json:"source"`          // 来源插件名称
	PanType       int    `json:"pan_type"`
	Size          string `json:"size,omitempty"`
	Time          string `json:"time,omitempty"`
	Content       string `json:"content,omitempty"` // 原始链接
	IsTransferred bool   `json:"is_transferred"`    // 是否已转存
}

// TransferResult 转存结果
type TransferResult struct {
	Success     bool   `json:"success"`
	Title       string `json:"title"`
	URL         string `json:"url"`          // 原始URL
	OriginalURL string `json:"original_url"` // 原始分享链接
	NewURL      string `json:"new_url"`      // 转存后的新URL
	ShareURL    string `json:"share_url"`    // 转存后的分享链接
	Password    string `json:"password"`     // 分享密码
	Fid         string `json:"fid"`          // 文件ID
	Message     string `json:"message"`      // 错误信息或提示
	PanType     int    `json:"pan_type"`     // 网盘类型
	ExpiredType int    `json:"expired_type"` // 过期类型: 0=永久 1=7天 2=1天
}

// PanType 网盘类型常量
const (
	PanTypeQuark   = 0 // 夸克
	PanTypeBaidu   = 2 // 百度
	PanTypeAliyun  = 3 // 阿里
	PanTypeUC      = 4 // UC
	PanTypeXunlei  = 5 // 迅雷
)

// GetPanTypeName 获取网盘类型名称
func GetPanTypeName(panType int) string {
	names := map[int]string{
		PanTypeQuark:  "夸克",
		PanTypeBaidu:  "百度",
		PanTypeAliyun: "阿里",
		PanTypeUC:     "UC",
		PanTypeXunlei: "迅雷",
	}
	if name, ok := names[panType]; ok {
		return name
	}
	return "未知"
}

// GetCloudType 将PanType转换为pansou的cloud_type
func GetCloudType(panType int) string {
	cloudTypes := map[int]string{
		PanTypeQuark:  "quark",
		PanTypeBaidu:  "baidu",
		PanTypeAliyun: "aliyun",
		PanTypeUC:     "uc",
		PanTypeXunlei: "xunlei",
	}
	if cloudType, ok := cloudTypes[panType]; ok {
		return cloudType
	}
	return "quark" // 默认夸克
}

// PansouRequest Pansou搜索请求
type PansouRequest struct {
	Keyword    string   `json:"kw"`
	Res        string   `json:"res"`
	CloudTypes []string `json:"cloud_types"`
	Src        string   `json:"src"`
}

// PansouResponse Pansou搜索响应
type PansouResponse struct {
	Code    int               `json:"code"`
	Message string            `json:"msg"`
	Data    []PansouResultRaw `json:"data"`
}

// PansouResultRaw Pansou原始结果
type PansouResultRaw struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Pwd       string `json:"pwd"`
	Source    string `json:"source"`
	CloudType string `json:"cloud_type"`
	Size      int64  `json:"size"`
	Time      string `json:"time"`
}

// ToSearchResult 转换为SearchResult
func (p *PansouResultRaw) ToSearchResult() SearchResult {
	return SearchResult{
		Title:    p.Title,
		URL:      p.URL,
		Password: p.Pwd,
		Source:   p.Source,
		PanType:  cloudTypeToPanType(p.CloudType),
		Size:     formatSize(p.Size),
		Time:     p.Time,
		Content:  p.URL,
	}
}

// cloudTypeToPanType 云盘类型字符串转PanType
func cloudTypeToPanType(cloudType string) int {
	typeMap := map[string]int{
		"quark":  PanTypeQuark,
		"baidu":  PanTypeBaidu,
		"aliyun": PanTypeAliyun,
		"uc":     PanTypeUC,
		"xunlei": PanTypeXunlei,
	}
	if panType, ok := typeMap[cloudType]; ok {
		return panType
	}
	return PanTypeQuark
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	if size <= 0 {
		return ""
	}
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)
	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
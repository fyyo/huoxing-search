package netdisk

import (
	"context"
	
	"xinyue-go/internal/model"
)

// Netdisk 网盘接口
type Netdisk interface {
	// Transfer 转存分享链接
	// shareURL: 分享链接
	// password: 提取码
	// expiredType: 过期类型 (2=临时2天, 其他=永久)
	Transfer(ctx context.Context, shareURL, password string, expiredType int) (*model.TransferResult, error)
	
	// GetName 获取网盘名称
	GetName() string
	
	// IsConfigured 检查是否已配置
	IsConfigured() bool
	
	// TestConnection 测试连接是否正常
	// 返回错误信息，nil表示连接成功
	TestConnection(ctx context.Context) error
	
	// DeleteDirectory 删除指定目录
	// dirPath: 目录路径
	DeleteDirectory(ctx context.Context, dirPath string) error
	
	// CreateDirectory 创建指定目录
	// dirPath: 目录路径
	CreateDirectory(ctx context.Context, dirPath string) error
}

// NetdiskManager 网盘管理器接口
type NetdiskManager interface {
	// GetClient 获取指定类型的网盘客户端
	GetClient(panType int) (Netdisk, error)
}
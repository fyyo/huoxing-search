package netdisk

import (
	"context"
	"fmt"

	"xinyue-go/internal/model"
	"xinyue-go/internal/netdisk/aliyun"
	"xinyue-go/internal/netdisk/baidu"
	"xinyue-go/internal/netdisk/quark"
	"xinyue-go/internal/netdisk/uc"
	"xinyue-go/internal/netdisk/xunlei"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/repository"
)

type netdiskManager struct {
	quark      Netdisk
	baidu      Netdisk
	aliyun     Netdisk
	uc         Netdisk
	xunlei     Netdisk
	configRepo repository.ConfigRepository
}

// NewNetdiskManager 创建网盘管理器 - 简化版本，只读取认证信息
func NewNetdiskManager(cfg *config.Config) NetdiskManager {
	configRepo := repository.NewConfigRepository()
	ctx := context.Background()

	// 从数据库读取基本认证配置
	quarkCookie, _ := configRepo.GetByName(ctx, "quark_cookie")
	baiduCookie, _ := configRepo.GetByName(ctx, "baidu_cookie")
	aliAuth, _ := configRepo.GetByName(ctx, "Authorization")
	ucCookie, _ := configRepo.GetByName(ctx, "uc_cookie")
	xunleiCookie, _ := configRepo.GetByName(ctx, "xunlei_cookie")

	return &netdiskManager{
		quark: quark.NewQuarkClient(
			getConfigValue(quarkCookie),
			configRepo,
		),
		baidu: baidu.NewBaiduClient(
			getConfigValue(baiduCookie),
			configRepo,
		),
		aliyun: aliyun.NewAliyunClient(
			getConfigValue(aliAuth),
			configRepo,
		),
		uc: uc.NewUCClient(
			getConfigValue(ucCookie),
			configRepo,
		),
		xunlei: xunlei.NewXunleiClient(
			getConfigValue(xunleiCookie),
			configRepo,
		),
		configRepo: configRepo,
	}
}

// getConfigValue 安全获取配置值
func getConfigValue(conf *model.Config) string {
	if conf == nil {
		return ""
	}
	return conf.Value
}

// GetClient 获取指定类型的网盘客户端
// ⚠️ 重要：每次调用都创建新的客户端实例，避免并发时Cookie相互覆盖
func (m *netdiskManager) GetClient(panType int) (Netdisk, error) {
	ctx := context.Background()
	
	var client Netdisk

	switch panType {
	case model.PanTypeQuark:
		// 每次都从数据库读取最新配置并创建新实例
		quarkCookie, _ := m.configRepo.GetByName(ctx, "quark_cookie")
		client = quark.NewQuarkClient(getConfigValue(quarkCookie), m.configRepo)
		
	case model.PanTypeBaidu:
		// 每次都从数据库读取最新配置并创建新实例
		baiduCookie, _ := m.configRepo.GetByName(ctx, "baidu_cookie")
		client = baidu.NewBaiduClient(getConfigValue(baiduCookie), m.configRepo)
		
	case model.PanTypeAliyun:
		// 每次都从数据库读取最新配置并创建新实例
		aliAuth, _ := m.configRepo.GetByName(ctx, "Authorization")
		client = aliyun.NewAliyunClient(getConfigValue(aliAuth), m.configRepo)
		
	case model.PanTypeUC:
		// 每次都从数据库读取最新配置并创建新实例
		ucCookie, _ := m.configRepo.GetByName(ctx, "uc_cookie")
		client = uc.NewUCClient(getConfigValue(ucCookie), m.configRepo)
		
	case model.PanTypeXunlei:
		// 每次都从数据库读取最新配置并创建新实例
		xunleiCookie, _ := m.configRepo.GetByName(ctx, "xunlei_cookie")
		client = xunlei.NewXunleiClient(getConfigValue(xunleiCookie), m.configRepo)
		
	default:
		return nil, fmt.Errorf("不支持的网盘类型: %d", panType)
	}

	if !client.IsConfigured() {
		return nil, fmt.Errorf("网盘未配置: %s", client.GetName())
	}

	return client, nil
}
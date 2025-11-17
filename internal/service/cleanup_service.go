package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"xinyue-go/internal/netdisk"
	"xinyue-go/internal/pkg/logger"
	"xinyue-go/internal/repository"
)

// CleanupService 清理服务接口
type CleanupService interface {
	CleanExpiredResources(ctx context.Context) error
	CleanExpiredResourcesWithFiles(ctx context.Context) error
	StartScheduledCleanup(ctx context.Context, interval time.Duration)
}

type cleanupService struct {
	sourceRepo     repository.SourceRepository
	configRepo     repository.ConfigRepository
	netdiskManager netdisk.NetdiskManager
}

// NewCleanupService 创建清理服务
func NewCleanupService(configRepo repository.ConfigRepository, netdiskManager netdisk.NetdiskManager) CleanupService {
	return &cleanupService{
		sourceRepo:     repository.NewSourceRepository(),
		configRepo:     configRepo,
		netdiskManager: netdiskManager,
	}
}

// CleanExpiredResources 清理过期的临时资源（仅数据库记录）
// 规则：
// - is_time=1 的资源为临时资源
// - 创建时间超过7天的临时资源将被删除
func (s *cleanupService) CleanExpiredResources(ctx context.Context) error {
	logger.Info("🧹 开始清理过期临时资源（仅数据库）")
	
	// 计算7天前的时间戳
	expiryTime := time.Now().AddDate(0, 0, -7).Unix()
	
	// 删除过期的临时资源
	count, err := s.sourceRepo.DeleteExpiredTemp(ctx, expiryTime)
	if err != nil {
		logger.Error("清理过期临时资源失败", zap.Error(err))
		return err
	}
	
	logger.Info("✅ 清理过期临时资源完成",
		zap.Int64("deleted_count", count),
		zap.Int64("expiry_time", expiryTime),
	)
	
	return nil
}

// CleanExpiredResourcesWithFiles 清理过期的临时资源（包括网盘文件）
// 规则：
// - is_time=1 的资源为临时资源
// - 创建时间超过7天的临时资源将被删除
// - 同时清理各网盘的临时目录文件
func (s *cleanupService) CleanExpiredResourcesWithFiles(ctx context.Context) error {
	logger.Info("🧹 开始清理过期临时资源（包括网盘文件）")
	
	// 1. 检查是否启用网盘文件清理
	enableNetdiskCleanup := s.shouldCleanNetdiskFiles(ctx)
	
	// 2. 如果启用，先清理网盘文件
	if enableNetdiskCleanup {
		if err := s.cleanNetdiskFiles(ctx); err != nil {
			logger.Error("清理网盘文件失败", zap.Error(err))
			// 继续执行数据库清理，不因网盘清理失败而中断
		}
	}
	
	// 3. 清理数据库记录
	return s.CleanExpiredResources(ctx)
}

// shouldCleanNetdiskFiles 检查是否应该清理网盘文件
func (s *cleanupService) shouldCleanNetdiskFiles(ctx context.Context) bool {
	conf, err := s.configRepo.GetByName(ctx, "delete_netdisk_files")
	if err != nil || conf == nil {
		return false // 默认不清理
	}
	return conf.Value == "1" || conf.Value == "true"
}

// cleanNetdiskFiles 清理各网盘的临时目录
func (s *cleanupService) cleanNetdiskFiles(ctx context.Context) error {
	logger.Info("🗑️ 开始清理网盘临时文件")
	
	// 网盘类型映射：0=夸克 2=百度 3=阿里 4=UC 5=迅雷
	netdiskConfigs := map[int]string{
		0: "quark_file_time",
		2: "baidu_file_time",
		3: "ali_file_time",
		4: "uc_file_time",
		5: "xunlei_file_time",
	}
	
	successCount := 0
	failCount := 0
	
	for panType, configKey := range netdiskConfigs {
		// 获取临时目录路径
		conf, err := s.configRepo.GetByName(ctx, configKey)
		if err != nil || conf == nil || conf.Value == "" || conf.Value == "0" {
			logger.Info("跳过网盘清理（未配置临时目录）",
				zap.Int("pan_type", panType),
				zap.String("config_key", configKey),
			)
			continue
		}
		
		tempDirPath := conf.Value
		
		// 获取网盘客户端
		client, err := s.netdiskManager.GetClient(panType)
		if err != nil {
			logger.Warn("获取网盘客户端失败",
				zap.Int("pan_type", panType),
				zap.Error(err),
			)
			failCount++
			continue
		}
		
		// 检查是否已配置
		if !client.IsConfigured() {
			logger.Info("跳过网盘清理（未配置）",
				zap.Int("pan_type", panType),
				zap.String("netdisk", client.GetName()),
			)
			continue
		}
		
		// 清理流程：删除临时目录 -> 重建空目录
		logger.Info("开始清理网盘临时目录",
			zap.String("netdisk", client.GetName()),
			zap.String("dir_path", tempDirPath),
		)
		
		// 1. 删除临时目录
		if err := client.DeleteDirectory(ctx, tempDirPath); err != nil {
			logger.Error("删除临时目录失败",
				zap.String("netdisk", client.GetName()),
				zap.String("dir_path", tempDirPath),
				zap.Error(err),
			)
			failCount++
			continue
		}
		
		logger.Info("✅ 临时目录已删除",
			zap.String("netdisk", client.GetName()),
			zap.String("dir_path", tempDirPath),
		)
		
		// 2. 重建空目录
		if err := client.CreateDirectory(ctx, tempDirPath); err != nil {
			logger.Error("重建临时目录失败",
				zap.String("netdisk", client.GetName()),
				zap.String("dir_path", tempDirPath),
				zap.Error(err),
			)
			failCount++
			continue
		}
		
		logger.Info("✅ 临时目录已重建",
			zap.String("netdisk", client.GetName()),
			zap.String("dir_path", tempDirPath),
		)
		
		successCount++
	}
	
	logger.Info("🎉 网盘文件清理完成",
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
	)
	
	if failCount > 0 {
		return fmt.Errorf("部分网盘清理失败: %d个成功, %d个失败", successCount, failCount)
	}
	
	return nil
}

// StartScheduledCleanup 启动定时清理任务
func (s *cleanupService) StartScheduledCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	logger.Info("⏰ 启动定时清理任务",
		zap.Duration("interval", interval),
	)
	
	// 立即执行一次清理（包括网盘文件）
	if err := s.CleanExpiredResourcesWithFiles(ctx); err != nil {
		logger.Error("首次清理失败", zap.Error(err))
	}
	
	// 定时执行
	for {
		select {
		case <-ctx.Done():
			logger.Info("⏹️ 停止定时清理任务")
			return
		case <-ticker.C:
			// 使用完整清理（包括网盘文件）
			if err := s.CleanExpiredResourcesWithFiles(ctx); err != nil {
				logger.Error("定时清理失败", zap.Error(err))
			}
		}
	}
}
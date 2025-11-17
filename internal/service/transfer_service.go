package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"xinyue-go/internal/model"
	"xinyue-go/internal/netdisk"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/pkg/logger"
	"xinyue-go/internal/repository"
)

// TransferService 转存服务接口
type TransferService interface {
	BatchTransfer(ctx context.Context, req *model.TransferRequest) (*model.TransferResponse, error)
	TransferAndSave(ctx context.Context, req *model.TransferRequest) (*model.TransferResponse, error)
}

type transferService struct {
	sourceRepo repository.SourceRepository
	netdisk    netdisk.NetdiskManager
	config     *config.Config
}

// NewTransferService 创建转存服务
func NewTransferService(cfg *config.Config) TransferService {
	return &transferService{
		sourceRepo: repository.NewSourceRepository(),
		netdisk:    netdisk.NewNetdiskManager(cfg),
		config:     cfg,
	}
}

// BatchTransfer 批量转存（实现PHP版本的两阶段处理逻辑）
// 阶段1: 转存前N条链接（MaxCount）
// 阶段2: 后M条链接不转存，仅验证有效性后返回原始链接（MaxDisplay - MaxCount）
func (s *transferService) BatchTransfer(ctx context.Context, req *model.TransferRequest) (*model.TransferResponse, error) {
	if len(req.Items) == 0 {
		return &model.TransferResponse{
			Total:   0,
			Success: 0,
			Failed:  0,
			Results: []model.TransferResult{},
		}, nil
	}

	// 设置默认值
	maxTransfer := req.MaxCount     // 最大转存数量
	maxDisplay := req.MaxDisplay    // 最大展示数量
	if maxTransfer <= 0 {
		maxTransfer = s.config.Transfer.MaxSuccess
	}
	if maxDisplay <= 0 {
		maxDisplay = maxTransfer // 如果未指定，则展示数量=转存数量
	}

	logger.Info("开始批量转存（两阶段处理）",
		zap.Int("total_items", len(req.Items)),
		zap.Int("pan_type", req.PanType),
		zap.Int("max_transfer", maxTransfer),     // 需要转存的数量
		zap.Int("max_display", maxDisplay),       // 总展示数量
	)

	allResults := make([]model.TransferResult, 0, maxDisplay)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 并发控制信号量
	semaphore := make(chan struct{}, s.config.Transfer.MaxConcurrent)
	transferredCount := 0    // 已转存成功的数量
	stopTransfer := false

	// 🔄 阶段1: 转存前 maxTransfer 条链接
	phase1Count := maxTransfer
	if phase1Count > len(req.Items) {
		phase1Count = len(req.Items)
	}

	logger.Info("📦 阶段1: 开始转存链接",
		zap.Int("count", phase1Count),
	)

	for i := 0; i < phase1Count; i++ {
		item := req.Items[i]
		
		// 检查是否已达到目标数量
		mu.Lock()
		if transferredCount >= maxTransfer || stopTransfer {
			mu.Unlock()
			logger.Info("达到转存目标数量，停止启动新转存",
				zap.Int("transferred_count", transferredCount),
				zap.Int("max_transfer", maxTransfer),
			)
			break
		}
		mu.Unlock()

		// 为每个链接创建独立的client实例
		client, err := s.netdisk.GetClient(req.PanType)
		if err != nil {
			logger.Error("获取网盘客户端失败",
				zap.String("title", item.Title),
				zap.Error(err),
			)
			continue
		}

		wg.Add(1)
		go func(searchItem model.SearchResult, netdiskClient netdisk.Netdisk) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 检查是否应该停止
			mu.Lock()
			shouldStop := stopTransfer || transferredCount >= maxTransfer
			mu.Unlock()
			if shouldStop {
				return
			}

			// 设置超时
			transferCtx, cancel := context.WithTimeout(ctx, s.config.Transfer.GetTimeout())
			defer cancel()

			// 执行转存
			result := s.transferSingleWithClient(transferCtx, searchItem, netdiskClient, req.PanType, req.ExpiredType)

			// 更新结果
			mu.Lock()
			defer mu.Unlock()

			if result.Success {
				if transferredCount < maxTransfer {
					allResults = append(allResults, result)
					transferredCount++
					logger.Info("✅ 阶段1转存成功",
						zap.String("title", result.Title),
						zap.String("new_url", result.NewURL),
						zap.Int("current", transferredCount),
						zap.Int("target", maxTransfer),
					)
					
					// 达到目标数量，设置停止标志
					if transferredCount >= maxTransfer {
						stopTransfer = true
					}
				}
			} else {
				logger.Warn("❌ 阶段1转存失败",
					zap.String("title", result.Title),
					zap.String("error", result.Message),
				)
			}
		}(item, client)
	}

	// 等待阶段1完成
	wg.Wait()

	logger.Info("📦 阶段1完成",
		zap.Int("transferred", transferredCount),
		zap.Int("target", maxTransfer),
	)

	// 🔍 阶段2: 处理剩余的链接（不转存，仅返回原始链接）
	phase2Count := maxDisplay - transferredCount
	if phase2Count > 0 && phase1Count < len(req.Items) {
		phase2Start := phase1Count
		phase2End := phase2Start + phase2Count
		if phase2End > len(req.Items) {
			phase2End = len(req.Items)
		}

		logger.Info("📋 阶段2: 添加未转存的原始链接",
			zap.Int("start", phase2Start),
			zap.Int("end", phase2End),
			zap.Int("count", phase2End-phase2Start),
		)

		for i := phase2Start; i < phase2End; i++ {
			item := req.Items[i]
			
			// 直接返回原始链接，不执行转存
			result := model.TransferResult{
				Success:     true,
				Title:       item.Title,
				URL:         item.URL,          // 原始URL
				OriginalURL: item.URL,
				NewURL:      item.URL,          // 未转存，使用原始URL
				ShareURL:    item.URL,
				Password:    item.Password,
				PanType:     item.PanType,
				Message:     "原始链接(未转存)",
			}
			
			allResults = append(allResults, result)
			
			logger.Debug("📋 添加原始链接",
				zap.String("title", item.Title),
				zap.String("url", item.URL),
			)
		}

		logger.Info("📋 阶段2完成",
			zap.Int("added", phase2End-phase2Start),
		)
	}

	response := &model.TransferResponse{
		Total:   len(allResults),
		Success: transferredCount,
		Failed:  maxTransfer - transferredCount,
		Results: allResults,
	}

	logger.Info("批量转存完成（两阶段）",
		zap.Int("total_display", response.Total),         // 总展示数量
		zap.Int("transferred", transferredCount),         // 实际转存数量
		zap.Int("original_links", len(allResults)-transferredCount), // 原始链接数量
	)

	return response, nil
}

// TransferAndSave 转存并保存到数据库
// ⚠️ 关键修改：只保存实际转存的链接，未转存的原始链接不保存到数据库
func (s *transferService) TransferAndSave(ctx context.Context, req *model.TransferRequest) (*model.TransferResponse, error) {
	// 执行转存（两阶段处理）
	resp, err := s.BatchTransfer(ctx, req)
	if err != nil {
		return nil, err
	}

	// 只保存实际转存成功的结果到数据库（阶段1的结果）
	// 阶段2的原始链接不保存到数据库
	if resp.Success > 0 {
		sources := make([]*model.Source, 0, resp.Success)
		now := time.Now().Unix()

		transferredCount := 0
		for _, result := range resp.Results {
			// 只保存实际转存的链接（Message不是"原始链接(未转存)"）
			if result.Success && result.Message != "原始链接(未转存)" {
				// ✅ 修复：根据用户请求的 expired_type 判断是否为临时资源
				// 而非使用网盘API返回值（百度网盘API永远返回0）
				// 1=永久 2=临时
				isTime := 0
				if req.ExpiredType == 2 {  // 用户选择了临时转存
					isTime = 1
				}
				
				logger.Debug("保存转存结果",
					zap.String("title", result.Title),
					zap.Int("req_expired_type", req.ExpiredType),
					zap.Int("api_expired_type", result.ExpiredType),
					zap.Int("is_time", isTime),
				)
				
				source := &model.Source{
					Title:      result.Title,
					URL:        result.NewURL,     // 转存后的新URL
					Content:    result.URL,        // 原始URL
					IsType:     result.PanType,
					Fid:        result.Fid,
					IsTime:     isTime,            // 根据用户选择设置（保持与PHP版本一致）
					Status:     1,
					CreateTime: now,
					UpdateTime: now,
				}
				sources = append(sources, source)
				transferredCount++
				
				// 达到实际转存数量后停止
				if transferredCount >= resp.Success {
					break
				}
			}
		}

		if len(sources) > 0 {
			if err := s.sourceRepo.BatchCreate(ctx, sources); err != nil {
				logger.Error("保存转存结果到数据库失败", zap.Error(err))
				// 不影响转存结果的返回
			} else {
				logger.Info("保存转存结果到数据库成功",
					zap.Int("count", len(sources)),
				)
			}
		}
	}

	return resp, nil
}

// transferSingleWithClient 使用指定的client实例进行单个转存
// ⚠️ 关键：使用传入的client实例，确保整个转存过程（verifyPassCode + getTransferParams + transfer）
// 使用同一个实例，保持Cookie状态（特别是BDCLND）的连续性
func (s *transferService) transferSingleWithClient(ctx context.Context, item model.SearchResult, client netdisk.Netdisk, panType int, expiredType int) model.TransferResult {
	result := model.TransferResult{
		Success: false,
		Title:   item.Title,
		URL:     item.URL,
		PanType: panType,
	}

	// 执行转存 - 使用传入的client实例
	// client内部会依次调用: verifyPassCode → getTransferParams → transferFile → createShare
	// Cookie状态（如BDCLND）在这些步骤间保持连续
	transferResult, err := client.Transfer(ctx, item.URL, item.Password, expiredType)
	if err != nil {
		result.Message = fmt.Sprintf("转存失败: %v", err)
		return result
	}

	result.Success = true
	result.NewURL = transferResult.ShareURL
	result.Fid = transferResult.Fid
	result.ExpiredType = transferResult.ExpiredType  // ← 设置网盘API返回的过期类型
	result.Message = "转存成功"

	return result
}
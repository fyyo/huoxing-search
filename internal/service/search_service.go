package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"huoxing-search/pansou/config"
	pansouModel "huoxing-search/pansou/model"
	"huoxing-search/pansou/plugin"
	pansouService "huoxing-search/pansou/service"
	"huoxing-search/pansou/util"
	"huoxing-search/pansou/util/cache"

	"huoxing-search/internal/model"
	"huoxing-search/internal/pkg/logger"
	"huoxing-search/internal/repository"
	
	// 导入所有Pansou插件以触发自动注册
	_ "huoxing-search/pansou/plugin/hdr4k"
	_ "huoxing-search/pansou/plugin/gying"
	_ "huoxing-search/pansou/plugin/pan666"
	_ "huoxing-search/pansou/plugin/hunhepan"
	_ "huoxing-search/pansou/plugin/jikepan"
	_ "huoxing-search/pansou/plugin/panwiki"
	_ "huoxing-search/pansou/plugin/pansearch"
	_ "huoxing-search/pansou/plugin/panta"
	_ "huoxing-search/pansou/plugin/qupansou"
	_ "huoxing-search/pansou/plugin/susu"
	_ "huoxing-search/pansou/plugin/thepiratebay"
	_ "huoxing-search/pansou/plugin/wanou"
	_ "huoxing-search/pansou/plugin/xuexizhinan"
	_ "huoxing-search/pansou/plugin/panyq"
	_ "huoxing-search/pansou/plugin/zhizhen"
	_ "huoxing-search/pansou/plugin/labi"
	_ "huoxing-search/pansou/plugin/muou"
	_ "huoxing-search/pansou/plugin/ouge"
	_ "huoxing-search/pansou/plugin/shandian"
	_ "huoxing-search/pansou/plugin/duoduo"
	_ "huoxing-search/pansou/plugin/huban"
	_ "huoxing-search/pansou/plugin/cyg"
	_ "huoxing-search/pansou/plugin/erxiao"
	_ "huoxing-search/pansou/plugin/miaoso"
	_ "huoxing-search/pansou/plugin/fox4k"
	_ "huoxing-search/pansou/plugin/pianku"
	_ "huoxing-search/pansou/plugin/clmao"
	_ "huoxing-search/pansou/plugin/wuji"
	_ "huoxing-search/pansou/plugin/cldi"
	_ "huoxing-search/pansou/plugin/xiaozhang"
	_ "huoxing-search/pansou/plugin/libvio"
	_ "huoxing-search/pansou/plugin/leijing"
	_ "huoxing-search/pansou/plugin/xb6v"
	_ "huoxing-search/pansou/plugin/xys"
	_ "huoxing-search/pansou/plugin/ddys"
	_ "huoxing-search/pansou/plugin/hdmoli"
	_ "huoxing-search/pansou/plugin/yuhuage"
	_ "huoxing-search/pansou/plugin/u3c3"
	_ "huoxing-search/pansou/plugin/javdb"
	_ "huoxing-search/pansou/plugin/clxiong"
	_ "huoxing-search/pansou/plugin/jutoushe"
	_ "huoxing-search/pansou/plugin/sdso"
	_ "huoxing-search/pansou/plugin/xiaoji"
	_ "huoxing-search/pansou/plugin/xdyh"
	_ "huoxing-search/pansou/plugin/haisou"
	_ "huoxing-search/pansou/plugin/bixin"
	_ "huoxing-search/pansou/plugin/nyaa"
	_ "huoxing-search/pansou/plugin/djgou"
	_ "huoxing-search/pansou/plugin/xinjuc"
	_ "huoxing-search/pansou/plugin/aikanzy"
	_ "huoxing-search/pansou/plugin/qupanshe"
	_ "huoxing-search/pansou/plugin/xdpan"
	_ "huoxing-search/pansou/plugin/discourse"
	_ "huoxing-search/pansou/plugin/yunsou"
	_ "huoxing-search/pansou/plugin/ahhhhfs"
	_ "huoxing-search/pansou/plugin/nsgame"
	_ "huoxing-search/pansou/plugin/quark4k"
	_ "huoxing-search/pansou/plugin/quarksoo"
	_ "huoxing-search/pansou/plugin/sousou"
	_ "huoxing-search/pansou/plugin/ash"
	_ "huoxing-search/pansou/plugin/qqpd"
	_ "huoxing-search/pansou/plugin/weibo"
)

// SearchService 搜索服务（集成Pansou）
type SearchService struct {
	configRepo      repository.ConfigRepository
	sourceRepo      repository.SourceRepository
	cacheRepo       repository.CacheRepository
	transferService TransferService
	pansouService   *pansouService.SearchService
	pluginManager   *plugin.PluginManager
	initialized     bool
}

// NewSearchService 创建搜索服务实例
func NewSearchService(configRepo repository.ConfigRepository, cacheRepo repository.CacheRepository, transferService TransferService) *SearchService {
	s := &SearchService{
		configRepo:      configRepo,
		sourceRepo:      repository.NewSourceRepository(),
		cacheRepo:       cacheRepo,
		transferService: transferService,
		initialized:     false,
	}
	
	// 异步初始化Pansou（避免阻塞启动）
	go s.initPansou()
	
	return s
}

// initPansou 初始化Pansou搜索引擎
func (s *SearchService) initPansou() error {
	// 初始化Pansou配置
	config.Init()
	
	// 初始化HTTP客户端
	util.InitHTTPClient()
	
	// 初始化缓存写入管理器
	cacheWriteManager, err := cache.NewDelayedBatchWriteManager()
	if err != nil {
		return fmt.Errorf("缓存写入管理器创建失败: %w", err)
	}
	if err := cacheWriteManager.Initialize(); err != nil {
		return fmt.Errorf("缓存写入管理器初始化失败: %w", err)
	}
	
	// 将缓存写入管理器注入到service包
	pansouService.SetGlobalCacheWriteManager(cacheWriteManager)
	
	// 延迟设置主缓存更新函数
	time.Sleep(100 * time.Millisecond)
	if mainCache := pansouService.GetEnhancedTwoLevelCache(); mainCache != nil {
		cacheWriteManager.SetMainCacheUpdater(func(key string, data []byte, ttl time.Duration) error {
			return mainCache.SetBothLevels(key, data, ttl)
		})
	}
	
	// 确保异步插件系统初始化
	plugin.InitAsyncPluginSystem()
	
	// 初始化插件管理器
	s.pluginManager = plugin.NewPluginManager()
	
	// 注册全局插件（根据配置过滤）
	if config.AppConfig.AsyncPluginEnabled {
		s.pluginManager.RegisterGlobalPluginsWithFilter(config.AppConfig.EnabledPlugins)
	}
	
	// 初始化Pansou搜索服务
	s.pansouService = pansouService.NewSearchService(s.pluginManager)
	s.initialized = true
	
	return nil
}

// Search 执行搜索 (实现: 优先本地 + 自动转存)
func (s *SearchService) Search(ctx context.Context, req model.SearchRequest) (*model.SearchResponse, error) {
	// 1. 参数验证
	if err := s.validateRequest(&req); err != nil {
		return nil, err
	}
	
	// 2. 关键词屏蔽检查
	if blocked, err := s.isKeywordBlocked(ctx, req.Keyword); err != nil {
		return nil, err
	} else if blocked {
		return &model.SearchResponse{
			Total:   0,
			Results: []model.SearchResult{},
			Message: "该关键词已被屏蔽",
		}, nil
	}
	
	// 📊 从配置表读取最大搜索结果数 (max_search_results)
	// 优先使用数据库配置，如果读取失败或为0，则使用请求参数或默认值
	maxSearchResults := 5  // 默认值
	if val, err := s.configRepo.GetInt(ctx, "max_search_results"); err == nil && val > 0 {
		maxSearchResults = val
	} else if req.MaxCount > 0 {
		maxSearchResults = req.MaxCount
	}
	
	// 📊 从配置表读取最大转存数量 (max_transfer_count)
	maxTransferCount := 2  // 默认值
	if val, err := s.configRepo.GetInt(ctx, "max_transfer_count"); err == nil && val > 0 {
		maxTransferCount = val
	}
	
	logger.Info("📊 配置参数",
		zap.Int("max_search_results", maxSearchResults),
		zap.Int("max_transfer_count", maxTransferCount),
	)
	
	// 🔍 第一步: 优先搜索本地数据库
	logger.Info("开始搜索本地数据库",
		zap.String("keyword", req.Keyword),
		zap.Int("pan_type", req.PanType),
	)
	
	localSources, err := s.sourceRepo.SearchByKeywordAndType(ctx, req.Keyword, req.PanType, maxSearchResults)
	if err == nil && len(localSources) > 0 {
		logger.Info("✅ 本地数据库命中",
			zap.Int("count", len(localSources)),
		)
		
		// 转换为SearchResult格式
		results := s.convertSourceToSearchResult(localSources)
		return &model.SearchResponse{
			Total:   len(results),
			Results: results,
			Message: "搜索成功(本地)",
		}, nil
	}
	
	logger.Info("本地数据库无结果,开始调用Pansou搜索")
	
	// 🌐 第二步: 本地无结果,调用Pansou搜索引擎
	// 等待Pansou初始化完成(最多等待5秒)
	for i := 0; i < 50 && !s.initialized; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	
	if !s.initialized {
		return nil, fmt.Errorf("Pansou搜索引擎初始化失败")
	}
	
	cloudType := model.GetCloudType(req.PanType)
	cloudTypes := []string{cloudType}
	
	// 调用Pansou搜索(获取20个结果用于转存)
	// 🔧 关键修复：让pansou使用所有可用插件
	pansouResp, err := s.pansouService.Search(
		req.Keyword,
		[]string{},                      // 🔧 改为空数组，让pansou使用默认配置（而非只用1个TG频道）
		config.AppConfig.DefaultConcurrency,
		false,                           // 不强制刷新，使用缓存
		"merged_by_type",                // 🔧 返回按类型合并的结果（包含多插件来源）
		"all",                           // ✅ 搜索所有来源（TG频道 + 插件）
		nil,                             // ✅ nil = 使用所有可用插件（50+插件）
		cloudTypes,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("Pansou搜索失败: %w", err)
	}
	
	// 转换Pansou结果 - 获取足够多的结果用于后续展示和转存
	// 获取 max(maxSearchResults * 4, 20) 个结果，确保有足够的资源进行转存筛选
	fetchCount := maxSearchResults * 4
	if fetchCount < 20 {
		fetchCount = 20
	}
	pansouResults := s.convertPansouResults(pansouResp, cloudType, fetchCount)
	
	if len(pansouResults) == 0 {
		logger.Info("Pansou搜索无结果")
		return &model.SearchResponse{
			Total:   0,
			Results: []model.SearchResult{},
			Message: "未找到相关资源",
		}, nil
	}
	
	// 📦 第三步: 尝试批量转存（如果转存服务可用且网盘已配置）
	logger.Info("📦 Pansou返回结果,检查是否可以转存",
		zap.Int("count", len(pansouResults)),
		zap.Int("target_display", maxSearchResults),
		zap.Int("target_transfer", maxTransferCount),
	)
	
	// 🔧 优先检查转存服务是否可用（微信公众号场景）
	if s.transferService == nil {
		logger.Info("⚠️ 转存服务未初始化（微信公众号禁用转存），跳过转存",
			zap.String("reason", "避免超过5秒响应限制"),
		)
		
		// 限制返回数量为配置的最大搜索结果数
		displayCount := maxSearchResults
		if displayCount > len(pansouResults) {
			displayCount = len(pansouResults)
		}
		
		finalResults := make([]model.SearchResult, 0, displayCount)
		for i := 0; i < displayCount; i++ {
			result := pansouResults[i]
			result.IsTransferred = false
			finalResults = append(finalResults, result)
		}
		
		return &model.SearchResponse{
			Total:   len(finalResults),
			Results: finalResults,
			Message: "搜索成功(原始链接，微信公众号)",
		}, nil
	}
	
	// 检查网盘是否已配置
	netdiskConfigured := s.isNetdiskConfigured(ctx, req.PanType)
	
	if !netdiskConfigured {
		// 网盘未配置，直接返回原始搜索结果
		logger.Info("⚠️ 网盘未配置，跳过转存，直接返回原始搜索结果",
			zap.Int("pan_type", req.PanType),
		)
		
		// 限制返回数量为配置的最大搜索结果数
		displayCount := maxSearchResults
		if displayCount > len(pansouResults) {
			displayCount = len(pansouResults)
		}
		
		finalResults := make([]model.SearchResult, 0, displayCount)
		for i := 0; i < displayCount; i++ {
			result := pansouResults[i]
			result.IsTransferred = false  // 标记为未转存
			finalResults = append(finalResults, result)
		}
		
		return &model.SearchResponse{
			Total:   len(finalResults),
			Results: finalResults,
			Message: "搜索成功(原始链接，网盘未配置)",
		}, nil
	}
	
	// 执行转存
	logger.Info("✅ 网盘已配置，开始批量转存（两阶段处理）",
		zap.Int("count", len(pansouResults)),
		zap.Int("target_transfer", maxTransferCount),      // 🔧 使用配置的转存数量
		zap.Int("target_display", maxSearchResults),       // 🔧 使用配置的展示数量
	)
	
	// 获取ExpiredType配置(1=永久, 2=临时)
	expiredType := 2 // 默认临时（is_time=1）
	if expiredConf, err := s.configRepo.GetInt(ctx, "default_expired_type"); err == nil && expiredConf > 0 {
		expiredType = expiredConf
	}
	
	transferReq := &model.TransferRequest{
		Items:       pansouResults,
		PanType:     req.PanType,
		MaxCount:    maxTransferCount,    // 🔧 使用配置表中的转存数量
		MaxDisplay:  maxSearchResults,    // 🔧 使用配置表中的展示数量
		ExpiredType: expiredType,         // 设置过期类型（临时资源）
	}
	
	transferResp, err := s.transferService.TransferAndSave(ctx, transferReq)
	if err != nil {
		// 转存失败，但不影响搜索功能，返回原始链接
		logger.Warn("转存失败，返回原始搜索结果", zap.Error(err))
		
		// 限制返回数量为配置的最大搜索结果数
		displayCount := maxSearchResults
		if displayCount > len(pansouResults) {
			displayCount = len(pansouResults)
		}
		
		finalResults := make([]model.SearchResult, 0, displayCount)
		for i := 0; i < displayCount; i++ {
			result := pansouResults[i]
			result.IsTransferred = false
			finalResults = append(finalResults, result)
		}
		
		return &model.SearchResponse{
			Total:   len(finalResults),
			Results: finalResults,
			Message: "搜索成功(原始链接，转存失败)",
		}, nil
	}
	
	if len(transferResp.Results) == 0 {
		logger.Warn("转存全部失败，返回原始搜索结果")
		
		// 限制返回数量为配置的最大搜索结果数
		displayCount := maxSearchResults
		if displayCount > len(pansouResults) {
			displayCount = len(pansouResults)
		}
		
		finalResults := make([]model.SearchResult, 0, displayCount)
		for i := 0; i < displayCount; i++ {
			result := pansouResults[i]
			result.IsTransferred = false
			finalResults = append(finalResults, result)
		}
		
		return &model.SearchResponse{
			Total:   len(finalResults),
			Results: finalResults,
			Message: "搜索成功(原始链接，转存全部失败)",
		}, nil
	}
	
	logger.Info("✅ 转存完成（两阶段）",
		zap.Int("total_display", len(transferResp.Results)),     // 总展示数量
		zap.Int("transferred", transferResp.Success),            // 实际转存数量
		zap.Int("original_links", len(transferResp.Results)-transferResp.Success), // 原始链接数量
	)
	
	// 📄 第四步: 将转存结果转换为搜索结果返回
	// 包含：转存后的新链接 + 未转存的原始链接
	finalResults := make([]model.SearchResult, 0, len(transferResp.Results))
	for i, tr := range transferResp.Results {
		if tr.Success {
			// 判断是转存链接还是原始链接
			isTransferred := tr.Message != "原始链接(未转存)"
			
			// 获取原始来源信息
			var sourceName string
			var sourceTime string
			if i < len(pansouResults) {
				sourceName = pansouResults[i].Source  // 来源插件名
				sourceTime = pansouResults[i].Time    // 来源时间
			}
			
			// 显示真实来源，而不是"已转存"
			if sourceName == "" {
				sourceName = "未知来源"
			}
			
			finalResults = append(finalResults, model.SearchResult{
				Title:         tr.Title,
				URL:           tr.NewURL,        // 转存后的新链接 或 原始链接
				Password:      tr.Password,
				Source:        sourceName,       // 显示真实来源（插件名）
				PanType:       tr.PanType,
				Time:          sourceTime,       // 显示原始时间
				Content:       tr.URL,           // 原始链接
				IsTransferred: isTransferred,    // 标记是否已转存
			})
		}
	}
	
	return &model.SearchResponse{
		Total:   len(finalResults),
		Results: finalResults,
		Message: fmt.Sprintf("搜索成功(已转存%d条,原始链接%d条)", transferResp.Success, len(finalResults)-transferResp.Success),
	}, nil
}

// convertSourceToSearchResult 将Source转换为SearchResult
func (s *SearchService) convertSourceToSearchResult(sources []*model.Source) []model.SearchResult {
	results := make([]model.SearchResult, 0, len(sources))
	for _, source := range sources {
		result := model.SearchResult{
			Title:    source.Title,
			URL:      source.URL,
			Password: "",
			Source:   "本地资源",
			PanType:  source.IsType,
			Content:  source.Content,
		}
		results = append(results, result)
	}
	return results
}

// convertPansouResults 转换Pansou搜索结果为xinyue格式
// 策略：从MergedByType中获取结果，这些结果已经按时间排序且来自不同插件
func (s *SearchService) convertPansouResults(pansouResp pansouModel.SearchResponse, cloudType string, maxCount int) []model.SearchResult {
	results := make([]model.SearchResult, 0)
	
	// 从MergedByType中提取指定网盘类型的链接
	// Pansou的MergedByType已经包含了来自多个插件的结果，按时间排序
	if mergedLinks, ok := pansouResp.MergedByType[cloudType]; ok {
		logger.Info("从MergedByType获取搜索结果",
			zap.Int("total", len(mergedLinks)),
			zap.String("cloud_type", cloudType),
			zap.Int("max_count", maxCount),
		)
		
		for i, link := range mergedLinks {
			if i >= maxCount {
				break
			}
			
			// 提取来源信息
			source := "未知"
			if strings.HasPrefix(link.Source, "tg:") {
				source = strings.TrimPrefix(link.Source, "tg:")
			} else if strings.HasPrefix(link.Source, "plugin:") {
				source = strings.TrimPrefix(link.Source, "plugin:")
			}
			
			// 格式化时间
			timeStr := ""
			if !link.Datetime.IsZero() {
				timeStr = link.Datetime.Format("2006-01-02")
			}
			
			result := model.SearchResult{
				Title:    link.Note,
				URL:      link.URL,
				Password: link.Password,
				Source:   source,  // 显示来源插件名
				PanType:  cloudTypeToPanType(cloudType),
				Time:     timeStr,
				Content:  link.URL,
			}
			
			results = append(results, result)
			
			// 记录每个结果的来源以便调试
			logger.Debug("添加搜索结果",
				zap.Int("index", i),
				zap.String("source", source),
				zap.String("title", link.Note),
			)
		}
		
		logger.Info("搜索结果转换完成",
			zap.Int("count", len(results)),
		)
	} else {
		logger.Warn("MergedByType中未找到指定网盘类型",
			zap.String("cloud_type", cloudType),
		)
	}
	
	return results
}

// cloudTypeToPanType 云盘类型字符串转PanType
func cloudTypeToPanType(cloudType string) int {
	typeMap := map[string]int{
		"quark":  model.PanTypeQuark,
		"baidu":  model.PanTypeBaidu,
		"aliyun": model.PanTypeAliyun,
		"uc":     model.PanTypeUC,
		"xunlei": model.PanTypeXunlei,
	}
	if panType, ok := typeMap[cloudType]; ok {
		return panType
	}
	return model.PanTypeQuark
}

// validateRequest 验证请求参数
func (s *SearchService) validateRequest(req *model.SearchRequest) error {
	if strings.TrimSpace(req.Keyword) == "" {
		return fmt.Errorf("搜索关键词不能为空")
	}
	
	if req.PanType < 0 || req.PanType > 5 {
		return fmt.Errorf("无效的网盘类型: %d", req.PanType)
	}
	
	return nil
}

// isKeywordBlocked 检查关键词是否被屏蔽
func (s *SearchService) isKeywordBlocked(ctx context.Context, keyword string) (bool, error) {
	banKeywords, err := s.configRepo.Get(ctx, model.ConfBanKeywords)
	if err != nil || banKeywords == "" {
		return false, nil
	}
	
	// 分割屏蔽关键词列表
	blocked := strings.Split(banKeywords, ",")
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	
	for _, bk := range blocked {
		bk = strings.ToLower(strings.TrimSpace(bk))
		if bk != "" && strings.Contains(keyword, bk) {
			return true, nil
		}
	}
	
	return false, nil
}

// ClearCache 清除搜索缓存
func (s *SearchService) ClearCache(ctx context.Context, keyword string, panType int) error {
	cacheKey := fmt.Sprintf("search:%s:%d", keyword, panType)
	return s.cacheRepo.Delete(ctx, cacheKey)
}

// isNetdiskConfigured 检查指定网盘是否已配置
func (s *SearchService) isNetdiskConfigured(ctx context.Context, panType int) bool {
	// 根据网盘类型获取对应的配置键名
	var configKey string
	switch panType {
	case model.PanTypeQuark:
		configKey = "quark_cookie"
	case model.PanTypeBaidu:
		configKey = "baidu_cookie"
	case model.PanTypeAliyun:
		configKey = "aliyun_refresh_token"
	case model.PanTypeUC:
		configKey = "uc_cookie"
	case model.PanTypeXunlei:
		configKey = "xunlei_cookie"
	default:
		return false
	}
	
	// 获取配置值
	value, err := s.configRepo.Get(ctx, configKey)
	if err != nil {
		logger.Debug("获取网盘配置失败",
			zap.String("config_key", configKey),
			zap.Error(err),
		)
		return false
	}
	
	// 检查配置值是否为空
	configured := strings.TrimSpace(value) != ""
	
	logger.Debug("网盘配置检查",
		zap.Int("pan_type", panType),
		zap.String("config_key", configKey),
		zap.Bool("configured", configured),
	)
	
	return configured
}

// ClearAllCache 清除所有搜索缓存
func (s *SearchService) ClearAllCache(ctx context.Context) error {
	return s.cacheRepo.DeletePattern(ctx, "search:*")
}
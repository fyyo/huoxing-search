// Package pansou 提供网盘搜索功能
package pansou

import (
	"huoxing-search/pansou/config"
	"huoxing-search/pansou/plugin"
	"huoxing-search/pansou/service"
	"huoxing-search/pansou/util"

	// 导入所有插件以触发自动注册
	_ "huoxing-search/pansou/plugin/ahhhhfs"
	_ "huoxing-search/pansou/plugin/aikanzy"
	_ "huoxing-search/pansou/plugin/ash"
	_ "huoxing-search/pansou/plugin/bixin"
	_ "huoxing-search/pansou/plugin/cldi"
	_ "huoxing-search/pansou/plugin/clmao"
	_ "huoxing-search/pansou/plugin/clxiong"
	_ "huoxing-search/pansou/plugin/cyg"
	_ "huoxing-search/pansou/plugin/ddys"
	_ "huoxing-search/pansou/plugin/discourse"
	_ "huoxing-search/pansou/plugin/djgou"
	_ "huoxing-search/pansou/plugin/duoduo"
	_ "huoxing-search/pansou/plugin/erxiao"
	_ "huoxing-search/pansou/plugin/fox4k"
	_ "huoxing-search/pansou/plugin/gying"
	_ "huoxing-search/pansou/plugin/haisou"
	_ "huoxing-search/pansou/plugin/hdr4k"
	_ "huoxing-search/pansou/plugin/hdmoli"
	_ "huoxing-search/pansou/plugin/huban"
	_ "huoxing-search/pansou/plugin/hunhepan"
	_ "huoxing-search/pansou/plugin/javdb"
	_ "huoxing-search/pansou/plugin/jikepan"
	_ "huoxing-search/pansou/plugin/jutoushe"
	_ "huoxing-search/pansou/plugin/labi"
	_ "huoxing-search/pansou/plugin/leijing"
	_ "huoxing-search/pansou/plugin/libvio"
	_ "huoxing-search/pansou/plugin/miaoso"
	_ "huoxing-search/pansou/plugin/muou"
	_ "huoxing-search/pansou/plugin/nsgame"
	_ "huoxing-search/pansou/plugin/nyaa"
	_ "huoxing-search/pansou/plugin/ouge"
	_ "huoxing-search/pansou/plugin/pan666"
	_ "huoxing-search/pansou/plugin/pansearch"
	_ "huoxing-search/pansou/plugin/panta"
	_ "huoxing-search/pansou/plugin/panwiki"
	_ "huoxing-search/pansou/plugin/panyq"
	_ "huoxing-search/pansou/plugin/pianku"
	_ "huoxing-search/pansou/plugin/qqpd"
	_ "huoxing-search/pansou/plugin/quark4k"
	_ "huoxing-search/pansou/plugin/quarksoo"
	_ "huoxing-search/pansou/plugin/qupanshe"
	_ "huoxing-search/pansou/plugin/qupansou"
	_ "huoxing-search/pansou/plugin/sdso"
	_ "huoxing-search/pansou/plugin/shandian"
	_ "huoxing-search/pansou/plugin/sousou"
	_ "huoxing-search/pansou/plugin/susu"
	_ "huoxing-search/pansou/plugin/thepiratebay"
	_ "huoxing-search/pansou/plugin/u3c3"
	_ "huoxing-search/pansou/plugin/wanou"
	_ "huoxing-search/pansou/plugin/weibo"
	_ "huoxing-search/pansou/plugin/wuji"
	_ "huoxing-search/pansou/plugin/xb6v"
	_ "huoxing-search/pansou/plugin/xdpan"
	_ "huoxing-search/pansou/plugin/xdyh"
	_ "huoxing-search/pansou/plugin/xiaoji"
	_ "huoxing-search/pansou/plugin/xiaozhang"
	_ "huoxing-search/pansou/plugin/xinjuc"
	_ "huoxing-search/pansou/plugin/xuexizhinan"
	_ "huoxing-search/pansou/plugin/xys"
	_ "huoxing-search/pansou/plugin/yuhuage"
	_ "huoxing-search/pansou/plugin/yunsou"
	_ "huoxing-search/pansou/plugin/zhizhen"
)

// SearchService 搜索服务实例
var SearchService *service.SearchService

// Init 初始化pansou搜索引擎
func Init() error {
	// 初始化配置
	config.Init()

	// 初始化HTTP客户端
	util.InitHTTPClient()

	// 初始化插件系统
	plugin.InitAsyncPluginSystem()

	// 创建插件管理器
	pluginManager := plugin.NewPluginManager()

	// 注册全局插件（根据配置过滤）
	if config.AppConfig.AsyncPluginEnabled {
		pluginManager.RegisterGlobalPluginsWithFilter(config.AppConfig.EnabledPlugins)
	}

	// 创建搜索服务
	SearchService = service.NewSearchService(pluginManager)

	return nil
}

// GetSearchService 获取搜索服务实例
func GetSearchService() *service.SearchService {
	return SearchService
}
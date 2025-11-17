// Package pansou 提供网盘搜索功能
package pansou

import (
	"xinyue-go/pansou/config"
	"xinyue-go/pansou/plugin"
	"xinyue-go/pansou/service"
	"xinyue-go/pansou/util"

	// 导入所有插件以触发自动注册
	_ "xinyue-go/pansou/plugin/ahhhhfs"
	_ "xinyue-go/pansou/plugin/aikanzy"
	_ "xinyue-go/pansou/plugin/ash"
	_ "xinyue-go/pansou/plugin/bixin"
	_ "xinyue-go/pansou/plugin/cldi"
	_ "xinyue-go/pansou/plugin/clmao"
	_ "xinyue-go/pansou/plugin/clxiong"
	_ "xinyue-go/pansou/plugin/cyg"
	_ "xinyue-go/pansou/plugin/ddys"
	_ "xinyue-go/pansou/plugin/discourse"
	_ "xinyue-go/pansou/plugin/djgou"
	_ "xinyue-go/pansou/plugin/duoduo"
	_ "xinyue-go/pansou/plugin/erxiao"
	_ "xinyue-go/pansou/plugin/fox4k"
	_ "xinyue-go/pansou/plugin/gying"
	_ "xinyue-go/pansou/plugin/haisou"
	_ "xinyue-go/pansou/plugin/hdr4k"
	_ "xinyue-go/pansou/plugin/hdmoli"
	_ "xinyue-go/pansou/plugin/huban"
	_ "xinyue-go/pansou/plugin/hunhepan"
	_ "xinyue-go/pansou/plugin/javdb"
	_ "xinyue-go/pansou/plugin/jikepan"
	_ "xinyue-go/pansou/plugin/jutoushe"
	_ "xinyue-go/pansou/plugin/labi"
	_ "xinyue-go/pansou/plugin/leijing"
	_ "xinyue-go/pansou/plugin/libvio"
	_ "xinyue-go/pansou/plugin/miaoso"
	_ "xinyue-go/pansou/plugin/muou"
	_ "xinyue-go/pansou/plugin/nsgame"
	_ "xinyue-go/pansou/plugin/nyaa"
	_ "xinyue-go/pansou/plugin/ouge"
	_ "xinyue-go/pansou/plugin/pan666"
	_ "xinyue-go/pansou/plugin/pansearch"
	_ "xinyue-go/pansou/plugin/panta"
	_ "xinyue-go/pansou/plugin/panwiki"
	_ "xinyue-go/pansou/plugin/panyq"
	_ "xinyue-go/pansou/plugin/pianku"
	_ "xinyue-go/pansou/plugin/qqpd"
	_ "xinyue-go/pansou/plugin/quark4k"
	_ "xinyue-go/pansou/plugin/quarksoo"
	_ "xinyue-go/pansou/plugin/qupanshe"
	_ "xinyue-go/pansou/plugin/qupansou"
	_ "xinyue-go/pansou/plugin/sdso"
	_ "xinyue-go/pansou/plugin/shandian"
	_ "xinyue-go/pansou/plugin/sousou"
	_ "xinyue-go/pansou/plugin/susu"
	_ "xinyue-go/pansou/plugin/thepiratebay"
	_ "xinyue-go/pansou/plugin/u3c3"
	_ "xinyue-go/pansou/plugin/wanou"
	_ "xinyue-go/pansou/plugin/weibo"
	_ "xinyue-go/pansou/plugin/wuji"
	_ "xinyue-go/pansou/plugin/xb6v"
	_ "xinyue-go/pansou/plugin/xdpan"
	_ "xinyue-go/pansou/plugin/xdyh"
	_ "xinyue-go/pansou/plugin/xiaoji"
	_ "xinyue-go/pansou/plugin/xiaozhang"
	_ "xinyue-go/pansou/plugin/xinjuc"
	_ "xinyue-go/pansou/plugin/xuexizhinan"
	_ "xinyue-go/pansou/plugin/xys"
	_ "xinyue-go/pansou/plugin/yuhuage"
	_ "xinyue-go/pansou/plugin/yunsou"
	_ "xinyue-go/pansou/plugin/zhizhen"
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